package pubsub

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/deixis/spine/contextutil"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
)

type MsgHandler func(context.Context, []byte)

type PubSub interface {
	Pub
	Sub

	// Start does the initialisation work to bootstrap a PubSub adapter.
	// For example, this function may open a connection, start an event loop, etc.
	Start(ctx context.Context) error
	// Drain signals to the pubsub client/server that inbound messages should
	// no longer be accepted, but outbound messages can still be delivered.
	Drain()
	// Close closes the client/server for both inbound/outbound messages
	Close() error
}

// Pub is the publish interface
type Pub interface {
	// Publish publishes data to the channel ch
	Publish(ctx context.Context, ch string, data []byte) error
}

// Sub is the subscribe interface
type Sub interface {
	// Subscribe subscribes the message handler h to the channel ch.
	// All subscriptions with the same q will form a queue group.
	// Each message will be delivered to only one subscriber per queue group.
	Subscribe(q, ch string, h MsgHandler) error
}

// Trace wraps `p` with a trace middleware
func Trace(p PubSub) PubSub {
	return &traceMiddleware{next: p}
}

type traceMiddleware struct {
	next PubSub
}

func (m *traceMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *traceMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	var span opentracing.Span
	span, ctx = tracing.StartSpanFromContext(ctx, "Pub "+ch)
	defer span.Finish()
	span.LogFields(
		olog.String("type", "pubsub"),
		olog.String("event", "publish"),
		olog.String("channel", ch),
	)

	return m.next.Publish(ctx, ch, data)
}

func (m *traceMiddleware) Subscribe(q, ch string, h MsgHandler) error {
	mh := func(ctx context.Context, data []byte) {
		var span opentracing.Span
		span, ctx = tracing.StartSpanFromContext(ctx, "Sub "+ch)
		defer span.Finish()
		span.LogFields(
			olog.String("type", "pubsub"),
			olog.String("event", "subscribe"),
			olog.String("queue", q),
			olog.String("channel", ch),
		)

		h(ctx, data)
	}
	return m.next.Subscribe(q, ch, mh)
}

func (m *traceMiddleware) Drain() {
	m.next.Drain()
}

func (m *traceMiddleware) Close() error {
	return m.next.Close()
}

// Log wraps `p` with trace logs
func Log(p PubSub) PubSub {
	return &logMiddleware{next: p}
}

type logMiddleware struct {
	next PubSub
}

func (m *logMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *logMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	err := m.next.Publish(ctx, ch, data)
	if err != nil {
		log.Warn(ctx, "pubsub.publish.err", "error publishing message",
			log.String("subject", ch),
			log.Int("data_len", len(data)),
			log.Error(err),
		)
		return err
	}

	log.Trace(ctx, "pubsub.publish.ok", "Message published",
		log.String("subject", ch),
		log.Int("data_len", len(data)),
	)
	return nil
}

func (m *logMiddleware) Subscribe(q, ch string, h MsgHandler) error {
	mh := func(ctx context.Context, data []byte) {
		startTime := time.Now()
		log.Trace(ctx, "pubsub.req.start", "Request start",
			log.String("queue", q),
			log.String("subject", ch),
			log.Int("data_len", len(data)),
		)

		h(ctx, data)

		log.Trace(ctx, "pubsub.req.end", "Request end",
			log.String("queue", q),
			log.String("subject", ch),
			log.Duration("duration", time.Now().Sub(startTime)),
		)
	}
	return m.next.Subscribe(q, ch, mh)
}

func (m *logMiddleware) Drain() {
	m.next.Drain()
}

func (m *logMiddleware) Close() error {
	return m.next.Close()
}

// Stats wraps `p` with stats
func Stats(p PubSub) PubSub {
	return &statsMiddleware{next: p}
}

type statsMiddleware struct {
	next PubSub
}

func (m *statsMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *statsMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	return m.next.Publish(ctx, ch, data)
}

func (m *statsMiddleware) Subscribe(q, ch string, h MsgHandler) error {
	tags := map[string]string{
		"subject": ch,
		"queue":   q,
	}

	mh := func(ctx context.Context, data []byte) {
		startTime := time.Now()
		stats.Inc(ctx, "pubsub.conc", tags)

		h(ctx, data)

		d := time.Now().Sub(startTime)
		stats.Histogram(ctx, "pubsub.call", 1, tags)
		stats.Timing(ctx, "pubsub.time", d, tags)
		stats.Dec(ctx, "pubsub.conc", tags)
	}
	return m.next.Subscribe(q, ch, mh)
}

func (m *statsMiddleware) Drain() {
	m.next.Drain()
}

func (m *statsMiddleware) Close() error {
	return m.next.Close()
}

// Recover wraps `p` with a middleware that recovers from panics in Subscribe
func Recover(p PubSub) PubSub {
	return &statsMiddleware{next: p}
}

type recoverMiddleware struct {
	next PubSub
}

func (m *recoverMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *recoverMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	return m.next.Publish(ctx, ch, data)
}

func (m *recoverMiddleware) Subscribe(q, ch string, h MsgHandler) error {
	mh := func(ctx context.Context, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				log.Err(ctx, "pubsub.panic", "Recovered from panic",
					log.String("queue", q),
					log.String("subject", ch),
					log.Object("error", r),
					log.String("stack", string(debug.Stack())),
				)
			}
		}()

		h(ctx, data)
	}
	return m.next.Subscribe(q, ch, mh)
}

func (m *recoverMiddleware) Drain() {
	m.next.Drain()
}

func (m *recoverMiddleware) Close() error {
	return m.next.Close()
}

type contextKey struct{}

var activePubContextKey = contextKey{}

// PubFromContext returns a `Pub` instance associated with `ctx`, or
// the local `Pub` if no instance could be found.
func PubFromContext(ctx contextutil.ValueContext) Pub {
	val := ctx.Value(activePubContextKey)
	if o, ok := val.(Pub); ok {
		return o
	}
	return NopPubSub()
}

// PubWithContext returns a copy of parent in which the `Pub` is stored
func PubWithContext(ctx context.Context, pub Pub) context.Context {
	return context.WithValue(ctx, activePubContextKey, pub)
}

var activeSubContextKey = contextKey{}

// SubFromContext returns a `Sub` instance associated with `ctx`, or
// the local `Sub` if no instance could be found.
func SubFromContext(ctx contextutil.ValueContext) Sub {
	val := ctx.Value(activeSubContextKey)
	if o, ok := val.(Sub); ok {
		return o
	}
	return NopPubSub()
}

// SubWithContext returns a copy of parent in which the `Sub` is stored
func SubWithContext(ctx context.Context, sub Sub) context.Context {
	return context.WithValue(ctx, activeSubContextKey, sub)
}

var activePubSubContextKey = contextKey{}

// FromContext returns a `PubSub` instance associated with `ctx`, or
// the local `Sub` if no instance could be found.
func FromContext(ctx contextutil.ValueContext) PubSub {
	val := ctx.Value(activePubSubContextKey)
	if o, ok := val.(PubSub); ok {
		return o
	}
	return NopPubSub()
}

// WithContext returns a copy of parent in which `PubSub`, `Pub`, and `Sub` are stored
func WithContext(ctx context.Context, ps PubSub) context.Context {
	return PubWithContext(
		SubWithContext(
			context.WithValue(ctx, activePubSubContextKey, ps),
			ps,
		),
		ps,
	)
}
