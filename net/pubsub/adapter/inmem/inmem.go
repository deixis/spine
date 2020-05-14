package inmem

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/deixis/spine/bg"
	"github.com/deixis/spine/config"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net"
	"github.com/deixis/spine/net/pubsub"
	"github.com/deixis/spine/tracing"
	"github.com/pkg/errors"
)

const (
	Name = "inmem"

	defaultBuffer = 50
)

type Inmem struct {
	mu    sync.RWMutex
	state uint32

	rootctx  context.Context
	log      log.Logger
	subs     map[string][]pubsub.MsgHandler
	channels map[string](chan *message)

	Config Config
}

type Config struct {
	Buffer int `toml:"buffer"`
}

func New(tree config.Tree) (pubsub.PubSub, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal pubsub.inmem config")
	}
	if config.Buffer == 0 {
		config.Buffer = defaultBuffer
	}

	return &Inmem{
		subs:     map[string][]pubsub.MsgHandler{},
		channels: map[string]chan *message{},
		Config:   *config,
	}, nil
}

func (ps *Inmem) Start(ctx context.Context) error {
	ps.rootctx = ctx
	ps.log = log.FromContext(ctx)
	atomic.StoreUint32(&ps.state, net.StateUp)
	return nil
}

func (ps *Inmem) Publish(
	ctx context.Context,
	ch string,
	data []byte,
) error {
	if !ps.isState(net.StateUp) {
		log.Warn(ps.rootctx, "pubsub.pub.draining", "PubSub is down or draining")
		return net.ErrDraining
	}

	// Publish message
	ps.load(ch) <- &message{
		Transit: lcontext.TransitFromContext(ctx),
		Data:    data,
	}
	return nil
}

func (ps *Inmem) Subscribe(q, ch string, h pubsub.MsgHandler) error {
	ps.mu.Lock()
	ps.subs[ch] = append(ps.subs[ch], h)
	ps.mu.Unlock()
	return nil
}

func (ps *Inmem) Drain() {
	atomic.StoreUint32(&ps.state, net.StateDrain)
	ps.mu.Lock()
	ps.subs = map[string][]pubsub.MsgHandler{}
	ps.mu.Unlock()
	atomic.StoreUint32(&ps.state, net.StateDown)
}

func (ps *Inmem) Close() error {
	atomic.StoreUint32(&ps.state, net.StateDown)
	return nil
}

// load returns a channel for the given and creates it if it does not exist
func (ps *Inmem) load(name string) chan *message {
	ps.mu.RLock()
	c, ok := ps.channels[name]
	ps.mu.RUnlock()
	if !ok {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		c = make(chan *message, ps.Config.Buffer)
		ps.channels[name] = c

		// Dispatch worker for that channel
		// New subscribes to the channel won't be updated on the worker
		// Normally they should all have been subscribed before the first load
		ps.log.Trace("pubsub.dispatch", "Dispatch worker",
			log.String("channel", name),
		)
		bg.Dispatch(ps.rootctx, &worker{
			rootctx: ps.rootctx, c: c, subscribers: ps.subs[name],
		})
	}
	return c
}

// isState checks the current server state
func (ps *Inmem) isState(state uint32) bool {
	return atomic.LoadUint32(&ps.state) == uint32(state)
}

type worker struct {
	rootctx     context.Context
	c           chan *message
	subscribers []pubsub.MsgHandler
	stop        chan bool
}

func (w *worker) Start() {
	w.stop = make(chan bool, 1)

	for {
		select {
		case <-w.stop:
			close(w.c)
			break
		case msg := <-w.c:
			if msg != nil {
				for _, call := range w.subscribers {
					// Create request root context
					ctx, cancel := context.WithCancel(w.rootctx)
					defer cancel()

					// Pick transit or create a new one, and attach it to context
					if msg.Transit != nil {
						ctx = lcontext.TransitWithContext(ctx, msg.Transit)
					} else {
						ctx, msg.Transit = lcontext.NewTransitWithContext(ctx)
					}

					// TODO: Create follow up transit

					// Attach contextualised services
					ctx = lcontext.WithTracer(ctx, tracing.FromContext(ctx))
					ctx = lcontext.WithLogger(ctx, log.FromContext(ctx))

					call(ctx, msg.Data)
				}
			}
		}
	}
}

func (w *worker) Stop() {
	w.stop <- true
}

type message struct {
	Transit lcontext.Transit
	Data    []byte
}
