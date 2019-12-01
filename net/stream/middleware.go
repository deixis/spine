package stream

import (
	"context"
	"runtime/debug"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
)

// Trace wraps `p` with a trace middleware
func Trace(s Stream) Stream {
	return &traceMiddleware{next: s}
}

type traceMiddleware struct {
	next Stream
}

func (m *traceMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *traceMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	var span opentracing.Span
	span, ctx = tracing.StartSpanFromContext(ctx, "Pub "+ch)
	defer span.Finish()
	span.LogFields(
		olog.String("type", "stream"),
		olog.String("event", "publish"),
		olog.String("channel", ch),
	)

	return m.next.Publish(ctx, ch, data)
}

func (m *traceMiddleware) Subscribe(
	q, ch string, h MsgHandler, opts ...SubscriptionOption,
) (Subscription, error) {
	mh := func(ctx context.Context, data []byte) error {
		var span opentracing.Span
		span, ctx = tracing.StartSpanFromContext(ctx, "Sub "+ch)
		defer span.Finish()
		span.LogFields(
			olog.String("type", "stream"),
			olog.String("event", "subscribe"),
			olog.String("queue", q),
			olog.String("channel", ch),
		)

		return h(ctx, data)
	}
	return m.next.Subscribe(q, ch, mh, opts...)
}

func (m *traceMiddleware) Drain() {
	m.next.Drain()
}

func (m *traceMiddleware) Close() error {
	return m.next.Close()
}

// Log wraps `p` with trace logs
func Log(s Stream) Stream {
	return &logMiddleware{next: s}
}

type logMiddleware struct {
	next Stream
}

func (m *logMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *logMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	err := m.next.Publish(ctx, ch, data)
	if err != nil {
		log.Warn(ctx, "stream.publish.err", "error publishing message",
			log.String("subject", ch),
			log.Int("data_len", len(data)),
			log.Error(err),
		)
		return err
	}

	log.Trace(ctx, "stream.publish.ok", "Message published",
		log.String("subject", ch),
		log.Int("data_len", len(data)),
	)
	return nil
}

func (m *logMiddleware) Subscribe(
	q, ch string, h MsgHandler, opts ...SubscriptionOption,
) (Subscription, error) {
	mh := func(ctx context.Context, data []byte) error {
		startTime := time.Now()
		log.Trace(ctx, "stream.req.start", "Request start",
			log.String("queue", q),
			log.String("subject", ch),
			log.Int("data_len", len(data)),
		)

		if err := h(ctx, data); err != nil {
			log.Trace(ctx, "stream.req.err", "Request error",
				log.String("queue", q),
				log.String("subject", ch),
				log.Duration("duration", time.Now().Sub(startTime)),
				log.Error(err),
			)
			return err
		}

		log.Trace(ctx, "stream.req.end", "Request end",
			log.String("queue", q),
			log.String("subject", ch),
			log.Duration("duration", time.Now().Sub(startTime)),
		)
		return nil
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
func Stats(s Stream) Stream {
	return &statsMiddleware{next: s}
}

type statsMiddleware struct {
	next Stream
}

func (m *statsMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *statsMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	return m.next.Publish(ctx, ch, data)
}

func (m *statsMiddleware) Subscribe(
	q, ch string, h MsgHandler, opts ...SubscriptionOption,
) (Subscription, error) {
	tags := map[string]string{
		"subject": ch,
		"queue":   q,
	}

	mh := func(ctx context.Context, data []byte) error {
		startTime := time.Now()
		stats.Inc(ctx, "stream.conc", tags)
		defer stats.Dec(ctx, "stream.conc", tags)

		if err := h(ctx, data); err != nil {
			return err
		}

		d := time.Now().Sub(startTime)
		stats.Histogram(ctx, "stream.call", 1, tags)
		stats.Timing(ctx, "stream.time", d, tags)
		return nil
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
func Recover(s Stream) Stream {
	return &statsMiddleware{next: s}
}

type recoverMiddleware struct {
	next Stream
}

func (m *recoverMiddleware) Start(ctx context.Context) error {
	return m.next.Start(ctx)
}

func (m *recoverMiddleware) Publish(ctx context.Context, ch string, data []byte) error {
	return m.next.Publish(ctx, ch, data)
}

func (m *recoverMiddleware) Subscribe(
	q, ch string, h MsgHandler, opts ...SubscriptionOption,
) (Subscription, error) {
	mh := func(ctx context.Context, data []byte) error {
		defer func() {
			if r := recover(); r != nil {
				log.Err(ctx, "stream.panic", "Recovered from panic",
					log.String("queue", q),
					log.String("subject", ch),
					log.Object("error", r),
					log.String("stack", string(debug.Stack())),
				)
			}
		}()

		return h(ctx, data)
	}
	return m.next.Subscribe(q, ch, mh, opts...)
}

func (m *recoverMiddleware) Drain() {
	m.next.Drain()
}

func (m *recoverMiddleware) Close() error {
	return m.next.Close()
}
