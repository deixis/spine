package http

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deixis/spine/bg"
	"github.com/deixis/spine/cache"
	"github.com/deixis/spine/config"
	scontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/disco"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net"
	"github.com/deixis/spine/schedule"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
	"github.com/gorilla/mux"
)

// A Server defines parameters for running a spine compatible HTTP server
// The zero value for Server is a valid configuration.
type Server struct {
	wg    sync.WaitGroup
	state uint32

	http http.Server

	endpoints   []Endpoint
	middlewares []Middleware

	certFile string
	keyFile  string

	config *config.Config
}

// NewServer creates a new server and attaches the default middlewares
func NewServer() *Server {
	s := &Server{}
	s.Append(mwDebug)
	s.Append(mwTrace)
	s.Append(mwStats)
	s.Append(mwLogging)
	return s
}

// HandleFunc registers a new function as an action on the given path and method
func (s *Server) HandleFunc(
	path,
	method string,
	f func(ctx context.Context, w ResponseWriter, r *Request),
) {
	s.HandleEndpoint(&stdEndpoint{
		path:       path,
		method:     method,
		handleFunc: f,
	})
}

// HandleStatic registers a new route on the given path with path prefix
// to serve static files from the provided root directory
func (s *Server) HandleStatic(
	path,
	root string,
	hook ...func(ctx context.Context, w ResponseWriter, r *Request, serveFile func()),
) {
	e := &fileEndpoint{
		path:        path,
		fileHandler: &fileHandler{root: http.Dir(root)},
	}
	if len(hook) > 0 {
		e.hook = hook[0]
	}
	s.HandleEndpoint(e)
}

// HandleEndpoint registers an endpoint.
// This is particularily useful for custom endpoint types
func (s *Server) HandleEndpoint(e Endpoint) {
	s.endpoints = append(s.endpoints, e)
}

// Append appends the given middleware to the call chain
func (s *Server) Append(m Middleware) {
	s.middlewares = append(s.middlewares, m)
}

// ActivateTLS activates TLS on this handler. That means only incoming HTTPS
// connections are allowed.
//
// If the certificate is signed by a certificate authority, the certFile should
// be the concatenation of the server's certificate, any intermediates,
// and the CA's certificate.
func (s *Server) ActivateTLS(certFile, keyFile string) {
	s.certFile = certFile
	s.keyFile = keyFile
}

// SetOptions changes the handler options
func (s *Server) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(s)
	}
}

// Serve starts serving HTTP requests (blocking call)
func (s *Server) Serve(ctx context.Context, addr string) error {
	cfg := config.Config{}
	if err := config.TreeFromContext(ctx).Unmarshal(&cfg); err != nil {
		return err
	}
	s.config = &cfg

	s.Append((&mwPanic{Panic: cfg.Request.Panic}).M)
	s.Append((&mwInterrupt{Panic: cfg.Request.Panic}).M)

	r := mux.NewRouter()
	for _, e := range s.endpoints {
		e.Attach(r, s.buildHandleFunc(ctx, e))
	}

	s.http.Addr = addr
	s.http.Handler = r

	tlsEnabled := s.certFile != "" && s.keyFile != ""
	log.FromContext(ctx).Trace(
		"s.http.listen",
		"Listening...",
		log.String("addr", addr),
		log.Bool("tls", tlsEnabled),
	)

	atomic.StoreUint32(&s.state, net.StateUp)
	var err error
	if tlsEnabled {
		err = s.http.ListenAndServeTLS(s.certFile, s.keyFile)
	}
	err = s.http.ListenAndServe()
	atomic.StoreUint32(&s.state, net.StateDown)

	if err == http.ErrServerClosed {
		// Suppress error caused by a server Shutdown or Close
		return nil
	}
	return err
}

// Drain puts the handler into drain mode. All new requests will be
// blocked with a 503 and it will block this call until all in-flight requests
// have been completed
func (s *Server) Drain() {
	atomic.StoreUint32(&s.state, net.StateDrain)
	s.wg.Wait()                           // Wait for all in-flight requests to complete
	s.http.Shutdown(context.Background()) // Then close all idle connections
}

// isState checks the current server state
func (s *Server) isState(state uint32) bool {
	return atomic.LoadUint32(&s.state) == uint32(state)
}

func (s *Server) buildHandleFunc(rootctx context.Context, e Endpoint) func(
	w http.ResponseWriter, r *http.Request) {

	serve := buildMiddlewareChain(s.middlewares, e)

	return func(w http.ResponseWriter, r *http.Request) {
		// Add to waitgroup for a graceful shutdown
		s.wg.Add(1)
		defer s.wg.Done()

		// Wrap net/http parameters
		res := &responseWriter{http: w}
		req := &Request{
			startTime: time.Now(),
			method:    e.Method(),
			path:      e.Path(),

			HTTP:   r,
			Params: mux.Vars(r),
		}

		// Ensure root ctx is still valid
		if err := rootctx.Err(); err != nil {
			log.FromContext(rootctx).Trace("http.stopped", "Server has stopped serving requests")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Attach app context services to request context
		// TODO: Merge with app context (like gRPC)
		ctx := r.Context()
		var cancel func()
		if s.config.Request.Timeout() > 0 {
			ctx, cancel = context.WithTimeout(ctx, s.config.Request.Timeout())
		} else {
			ctx, cancel = context.WithCancel(ctx)
		}
		defer cancel()

		// Attach app context services to request context
		ctx = config.TreeWithContext(ctx, config.TreeFromContext(rootctx))
		ctx = log.WithContext(ctx, log.FromContext(rootctx))
		ctx = stats.WithContext(ctx, stats.FromContext(rootctx))
		ctx = bg.RegWithContext(ctx, bg.RegFromContext(rootctx))
		ctx = tracing.WithContext(ctx, tracing.FromContext(rootctx))
		ctx = disco.AgentWithContext(ctx, disco.AgentFromContext(rootctx))
		ctx = schedule.SchedulerWithContext(ctx, schedule.SchedulerFromContext(rootctx))
		ctx = cache.WithContext(ctx, cache.FromContext(rootctx))

		// Decode context
		if s.config.Request.AllowContext {
			rctx, err := decodeContext(ctx, req.HTTP)
			if err != nil {
				log.FromContext(ctx).Warning(
					"http.context.decode.err",
					"Cannot decode context",
					log.Error(err),
				)
				w.WriteHeader(StatusBadRequest)
				return
			}
			ctx = rctx
		} else {
			ctx, _ = scontext.NewTransitWithContext(ctx)
		}

		// Attach contextualised services
		ctx = scontext.WithTracer(ctx, tracing.FromContext(ctx))
		ctx = scontext.WithLogger(ctx, log.FromContext(ctx))

		// Attach new context back to the HTTP request
		req.HTTP = req.HTTP.WithContext(ctx)

		if s.isState(net.StateDrain) {
			log.FromContext(ctx).Trace("http.draining", "Handler is draining")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Handle request
		serve(ctx, res, req)
	}
}
