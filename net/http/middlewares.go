package http

import (
	"context"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
)

// Middleware is a function called on the HTTP stack before an action
type Middleware func(ServeFunc) ServeFunc

func buildMiddlewareChain(l []Middleware, e Endpoint) ServeFunc {
	f := func(ctx context.Context, w ResponseWriter, r *Request) {
		e.Serve(ctx, w, r)
	}
	if len(l) == 0 {
		return f
	}

	c := f
	for i := len(l) - 1; i >= 0; i-- {
		c = l[i](c)
	}
	return c
}

// mwDebug adds useful debugging information to the response header
func mwDebug(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		tr := lcontext.TransitFromContext(ctx)
		w.Header().Add("Request-Id", tr.UUID())
		next(ctx, w, r)
	}
}

// mwTrace traces requests with the context `Tracer`
func mwTrace(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		var span opentracing.Span
		span, ctx = tracing.StartSpanFromContext(ctx, r.method+" "+r.path)
		defer span.Finish()
		span.LogFields(
			olog.String("event", "start"),
			olog.String("type", "http"),
			olog.String("method", r.method),
			olog.String("path", r.path),
			olog.String("startTime", r.startTime.Format(time.RFC3339Nano)),
		)

		// Next middleware
		next(ctx, w, r)

		span.LogFields(
			olog.String("event", "end"),
			olog.Int("status", w.Code()),
		)
	}
}

// mwLogging logs information about HTTP requests/responses
func mwLogging(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		logger := log.FromContext(ctx)
		logger.Trace("h.http.req.start", "Request start",
			log.String("method", r.method),
			log.String("path", r.path),
			log.String("user_agent", r.HTTP.Header.Get("User-Agent")),
		)

		next(ctx, w, r)

		logger.Trace("h.http.req.end", "Request end",
			log.String("method", r.method),
			log.String("path", r.path),
			log.Int("status", w.Code()),
			log.Duration("duration", time.Since(r.startTime)),
		)
	}
}

// mwStats sends the request/response stats
func mwStats(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		stats := stats.FromContext(ctx)
		tags := map[string]string{
			"method": r.method,
			"path":   r.path,
		}
		stats.Inc("http.conc", tags)

		// Next middleware
		next(ctx, w, r)

		tags["status"] = strconv.Itoa(w.Code())
		stats.Histogram("http.call", 1, tags)
		stats.Timing("http.time", time.Since(r.startTime), tags)
		stats.Dec("http.conc", tags)
	}
}

// mwPanic catches panic and recover
type mwPanic struct {
	Panic bool
}

func (mw *mwPanic) M(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		// Wrap call to the next middleware
		func() {
			defer func() {
				if mw.Panic {
					return
				}
				if recover := recover(); recover != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Err(ctx, "http.mw.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
				}
			}()

			next(ctx, w, r)
		}()
	}
}

// mwInterrupt interupts requests when the context cancels
type mwInterrupt struct {
	Panic bool
}

func (mw *mwInterrupt) M(next ServeFunc) ServeFunc {
	return func(ctx context.Context, w ResponseWriter, r *Request) {
		res := make(chan struct{}, 1)
		rec := make(chan interface{}, 1)

		go func() {
			defer func() {
				if mw.Panic {
					return
				}
				if recover := recover(); recover != nil {
					log.Err(ctx, "http.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
					rec <- recover
				}
			}()

			next(ctx, w, r)
			res <- struct{}{}
		}()

		select {
		case <-res:
			// OK
		case <-rec:
			// action panicked
			w.WriteHeader(http.StatusInternalServerError)
		case <-ctx.Done():
			log.Warn(ctx, "http.interrupt", "Request cancelled or timed out", log.Error(ctx.Err()))
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	}
}
