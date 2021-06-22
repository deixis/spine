package spine

import (
	"context"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/deixis/spine/bg"
	"github.com/deixis/spine/cache"
	acache "github.com/deixis/spine/cache/adapter"
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/disco"
	adisco "github.com/deixis/spine/disco/adapter"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/log/logger"
	"github.com/deixis/spine/net"
	"github.com/deixis/spine/net/pubsub"
	apubsub "github.com/deixis/spine/net/pubsub/adapter"
	"github.com/deixis/spine/net/stream"
	astream "github.com/deixis/spine/net/stream/adapter"
	"github.com/deixis/spine/schedule"
	aschedule "github.com/deixis/spine/schedule/adapter"
	"github.com/deixis/spine/stats"
	astats "github.com/deixis/spine/stats/adapter"
	"github.com/deixis/spine/tracing"
	atracing "github.com/deixis/spine/tracing/adapter"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

const (
	down uint32 = iota
	up
	drain
)

// App is the core structure for a new service
type App struct {
	mu     sync.Mutex
	ready  *sync.Cond
	ctx    context.Context
	cancel context.CancelFunc

	service    string
	config     config.Config
	configTree config.Tree
	state      uint32
	stopc      chan struct{}

	servers       *net.Reg
	registrations []*disco.Registration

	// services
	bg       *bg.Reg
	log      log.Logger
	stats    stats.Stats
	tracer   tracing.Tracer
	disco    disco.Agent
	cache    cache.Cache
	schedule schedule.Scheduler
	pubsub   pubsub.PubSub
	stream   stream.Stream

	drainHandlers []func(context.Context)
}

// New creates a new App and returns it
func New(service string, appConfig interface{}) (*App, error) {
	configStore, err := config.NewStore(os.Getenv("CONFIG_URI"))
	if err != nil {
		return nil, errors.Wrap(err, "error creating config store")
	}

	r, err := configStore.Load()
	if err != nil {
		return nil, errors.Wrap(err, "error loading load config")
	}
	defer r.Close()

	return NewWithConfig(service, r, appConfig)
}

// NewWithConfig creates a new App with a custom configuration
func NewWithConfig(
	service string, r io.Reader, appConfig interface{},
) (a *App, err error) {
	configTree, err := config.LoadTree(r)
	if err != nil {
		return nil, errors.Wrap(err, "error loading config tree")
	}

	err = configTree.Get("app").Unmarshal(appConfig)
	if err != nil {
		return nil, errors.Wrap(err, "annot unmarshal app config")
	}

	// Build app struct
	lock := &sync.Mutex{}
	lock.Lock()
	ready := sync.NewCond(lock)
	ctx, cancelFunc := context.WithCancel(context.Background())
	a = &App{
		ready:      ready,
		ctx:        ctx,
		cancel:     cancelFunc,
		service:    service,
		stopc:      make(chan struct{}),
		configTree: configTree,
	}

	err = a.configTree.Unmarshal(&a.config)
	if err != nil {
		return nil, errors.Wrap(err, "annot unmarshal core config")
	}
	a.ctx = config.TreeWithContext(a.ctx, a.configTree)

	// Set up services
	a.log, err = logger.New(service, a.configTree.Get("log"))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising logger")
	}
	a.log = a.log.With(
		log.String("service", service),
		log.String("node", a.config.Node),
		log.String("version", a.config.Version),
		log.String("log_type", "A"),
	)
	a.ctx = log.WithContext(a.ctx, a.log)

	a.stats, err = astats.New(a.configTree.Get("stats"))
	switch err {
	case astats.ErrEmptyConfig:
		a.stats = stats.NopStats()
		fallthrough
	case nil:
		a.stats = a.stats.With(map[string]string{
			"service": service,
			"node":    a.config.Node,
			"version": a.config.Version,
		})
		a.stats = a.stats.Log(a.log)
		a.ctx = stats.WithContext(a.ctx, a.stats)
	default:
		return nil, errors.Wrap(err, "error initialising stats")
	}

	a.bg = bg.NewReg(service, a)
	a.ctx = bg.RegWithContext(a.ctx, a.bg)

	a.tracer, err = atracing.New(
		a.configTree.Get("tracing"),
		tracing.WithLogger(a.log),
		tracing.WithStats(a.stats),
	)
	switch err {
	case atracing.ErrEmptyConfig:
		a.tracer = opentracing.GlobalTracer()
		fallthrough
	case nil:
		a.ctx = tracing.WithContext(a.ctx, a.tracer)
	default:
		return nil, errors.Wrap(err, "error initialising tracer")
	}

	a.disco, err = adisco.New(a.configTree.Get("disco"))
	switch err {
	case adisco.ErrEmptyConfig:
		a.disco = disco.NewLocalAgent()
		fallthrough
	case nil:
		a.ctx = disco.AgentWithContext(a.ctx, a.disco)
	default:
		return nil, errors.Wrap(err, "error initialising disco agent")
	}

	a.schedule, err = aschedule.New(a.configTree.Get("schedule"))
	switch err {
	case aschedule.ErrEmptyConfig:
		a.schedule = schedule.NopScheduler()
		fallthrough
	case nil:
		a.ctx = schedule.SchedulerWithContext(a.ctx, a.schedule)
	default:
		return nil, errors.Wrap(err, "error initialising scheduler")
	}

	a.cache, err = acache.New(a.configTree.Get("cache"))
	switch err {
	case acache.ErrEmptyConfig:
		a.cache = cache.NopCache()
		fallthrough
	case nil:
		a.ctx = cache.WithContext(a.ctx, a.cache)
	default:
		return nil, errors.Wrap(err, "error initialising cache")
	}

	a.pubsub, err = apubsub.New(a.configTree.Get("net").Get("pubsub"))
	switch err {
	case apubsub.ErrEmptyConfig:
		a.pubsub = pubsub.NopPubSub()
		fallthrough
	case nil:
		a.pubsub = pubsub.Trace(a.pubsub)
		a.pubsub = pubsub.Stats(a.pubsub)
		a.pubsub = pubsub.Log(a.pubsub)
		a.pubsub = pubsub.Recover(a.pubsub)
		a.ctx = pubsub.WithContext(a.ctx, a.pubsub)
	default:
		return nil, errors.Wrap(err, "error initialising net/pubsub")
	}

	a.stream, err = astream.New(a.configTree.Get("net").Get("stream"))
	switch err {
	case astream.ErrEmptyConfig:
		a.stream = stream.NopStream()
		fallthrough
	case nil:
		a.stream = stream.Trace(a.stream)
		a.stream = stream.Stats(a.stream)
		a.stream = stream.Log(a.stream)
		a.stream = stream.Recover(a.stream)
		a.ctx = stream.WithContext(a.ctx, a.stream)
	default:
		return nil, errors.Wrap(err, "error initialising net/stream")
	}

	// Trap OS signals
	go trapSignals(a)

	// Create net registry to register request handlers (HTTP, gRPC, PubSub, ...)
	a.servers = net.NewReg(a.log)

	// Start background services
	if err := a.BG().Dispatch(a.stats); err != nil {
		return nil, err
	}
	if err := a.BG().Dispatch(&hearbeat{app: a}); err != nil {
		return nil, err
	}
	if err := a.schedule.Start(a); err != nil {
		return nil, errors.Wrap(err, "error starting scheduler")
	}
	if err := a.pubsub.Start(a); err != nil {
		return nil, errors.Wrap(err, "error starting pubsub")
	}
	if err := a.stream.Start(a); err != nil {
		return nil, errors.Wrap(err, "error starting stream")
	}
	return a, nil
}

// Serve allows handlers to serve requests and blocks the call
func (a *App) Serve() error {
	defer func() {
		if recover := recover(); recover != nil {
			a.Error("spine.serve.panic", "App panic",
				log.Object("err", recover),
				log.String("stack", string(debug.Stack())),
			)

			a.Close()
			panic(recover)
		}
	}()

	if !a.isState(down) {
		a.Warning("spine.serve.state", "Server is not in down state",
			log.Uint("state", uint(a.state)),
		)
	}

	a.Trace("spine.serve", "Start serving...")

	err := a.servers.Serve(a)
	if err != nil {
		a.Error("spine.serve.error", "Error with handler.Serve (%s)",
			log.Error(err),
		)
		return err
	}

	for _, reg := range a.registrations {
		_, err := a.disco.Register(a, reg)
		if err != nil {
			return errors.Wrapf(err, "error registering service <%s>", reg.Name)
		}
	}

	// Notify all callees that the app is up and running
	a.ready.Broadcast()

	atomic.StoreUint32(&a.state, up)
	<-a.stopc
	return nil
}

// Ready holds the callee until the app is fully operational
func (a *App) Ready() {
	a.ready.Wait()
}

func (a *App) Service() string {
	return a.service
}

func (a *App) L() log.Logger {
	return a.log
}

func (a *App) Stats() stats.Stats {
	return a.stats
}

func (a *App) Tracer() tracing.Tracer {
	return a.tracer
}

func (a *App) Config() *config.Config {
	return &a.config
}

func (a *App) ConfigTree() config.Tree {
	return a.configTree
}

func (a *App) BG() *bg.Reg {
	return a.bg
}

func (a *App) Cache() cache.Cache {
	return a.cache
}

// Disco returns the active service discovery agent.
//
// When service discovery is disabled, it will return a local agent that acts
// like a regular service discovery agent, expect that it only registers local
// services.
func (a *App) Disco() disco.Agent {
	return a.disco
}

func (a *App) Scheduler() schedule.Scheduler {
	return a.schedule
}

// Drain notify all handlers to enter in draining mode. It means they are no
// longer accepting new requests, but they can finish all in-flight requests
func (a *App) Drain() bool {
	a.mu.Lock()
	if !a.isState(up) {
		a.mu.Unlock()
		return false
	}
	atomic.StoreUint32(&a.state, drain)
	a.mu.Unlock()

	a.Trace("spine.drain", "Start draining...")

	// Notify all handlers
	for _, h := range a.drainHandlers {
		h(a)
	}
	// Block all new requests and drain in-flight requests
	a.servers.Drain()
	a.Trace("spine.drain.schedule", "Start draining schedule...")
	a.schedule.Drain()
	a.Trace("spine.drain.stream", "Start draining stream...")
	a.stream.Drain()
	a.Trace("spine.drain.pubsub", "Start draining pubsub...")
	a.pubsub.Drain()
	a.Trace("spine.drain.bg", "Start draining background jobs...")
	a.bg.Drain()
	a.Trace("spine.drain.ok", "Drained")
	return true
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first draining all handlers, then
// draining the main context, and finally shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
func (a *App) Shutdown() {
	a.Trace("spine.shutdown", "Gracefully shutting down...")
	a.disco.Leave(a)
	if !a.Drain() {
		a.Trace("spine.shutdown.abort", "Server already draining")
		return
	}
	a.close()
}

// Close immediately closes the server and any in-flight request or background
// job will be left unfinished.
// For a graceful shutdown, use Shutdown.
func (a *App) Close() error {
	a.Trace("spine.close", "Closing immediately!")
	a.disco.Leave(a)
	a.close()
	return nil
}

func (a *App) close() {
	a.schedule.Close()
	if c, ok := a.tracer.(io.Closer); ok {
		c.Close()
	}
	a.log.Close()

	select {
	case a.stopc <- struct{}{}:
	default:
	}
	atomic.StoreUint32(&a.state, down)
}

// RegisterServer adds the given server to the list of managed servers
func (a *App) RegisterServer(addr string, s net.Server) {
	a.servers.Add(addr, s)
}

// ServiceRegistration contains info to register a service
type ServiceRegistration struct {
	// ID is the service instance unique identifier (optional)
	ID string
	// Name is the service identifier
	Name string
	// Host is the interface on which the server runs.
	// Service discovery can override this value.
	Host string
	// Port is the port number
	Port uint16
	// Server is the server that provides the registered service
	Server net.Server
	// Tags for that service (versioning, blue-green, whatever)
	Tags []string
}

// RegisterService adds the server to the list of managed servers and registers
// it to service discovery
func (a *App) RegisterService(r *ServiceRegistration) {
	a.servers.Add(net.JoinHostPort(r.Host, strconv.Itoa(int(r.Port))), r.Server)

	a.registrations = append(a.registrations, &disco.Registration{
		ID:   r.ID,
		Name: r.Name,
		Addr: r.Host,
		Port: r.Port,
		Tags: append(r.Tags, a.service),
	})
}

// RegisterDrainHandler registers a handler that is called when the app starts
// draining
func (a *App) RegisterDrainHandler(h func(ctx context.Context)) {
	a.drainHandlers = append(a.drainHandlers, h)
}

// isState checks the current app state
func (a *App) isState(state uint32) bool {
	return atomic.LoadUint32(&a.state) == uint32(state)
}

// Trace implements log.Logger
func (a *App) Trace(tag, msg string, fields ...log.Field) {
	a.log.Trace(tag, msg, fields...)
}

// Warning implements log.Logger
func (a *App) Warning(tag, msg string, fields ...log.Field) {
	a.log.Warning(tag, msg, fields...)
}

// Error implements log.Logger
func (a *App) Error(tag, msg string, fields ...log.Field) {
	a.log.Error(tag, msg, fields...)
}

// Deadline implements context.Context
func (a *App) Deadline() (deadline time.Time, ok bool) {
	return a.ctx.Deadline()
}

// Done implements context.Context
func (a *App) Done() <-chan struct{} {
	return a.ctx.Done()
}

// Err implements context.Context
func (a *App) Err() error {
	return a.ctx.Err()
}

// Value implements context.Context
func (a *App) Value(key interface{}) interface{} {
	return a.ctx.Value(key)
}

func trapSignals(app *App) {
	ch := make(chan os.Signal, 10)
	signals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	}
	signal.Notify(ch, signals...)

	logger := app.L()
	for {
		sig := <-ch
		n, _ := sig.(syscall.Signal)
		logger.Trace("spine.signal", "Signal trapped",
			log.String("sig", sig.String()),
			log.Int("n", int(n)),
		)

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			app.Shutdown()
			signal.Stop(ch)
			return
		case syscall.SIGQUIT, syscall.SIGKILL:
			app.Close()
			signal.Stop(ch)
			return
		default:
			logger.Error("spine.signal.unhandled", "Unhandled signal")
			os.Exit(128 + int(n))
			return
		}
	}
}
