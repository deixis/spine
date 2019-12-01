// Package jaeger wraps the Jaeger tracer
package jaeger

import (
	"fmt"
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	jaegermetrics "github.com/uber/jaeger-lib/metrics"
)

const Name = "jaeger"

func New(tree config.Tree, o ...tracing.TracerOption) (tracing.Tracer, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal tracing.jaeger config")
	}

	jconfig := jaegerconfig.Configuration{
		ServiceName: config.ServiceName,
		Disabled:    config.Disabled,
		RPCMetrics:  config.RPCMetrics,
	}
	jconfig.Tags = make([]opentracing.Tag, 0, len(config.Tags))
	for k, v := range config.Tags {
		jconfig.Tags = append(jconfig.Tags, opentracing.Tag{Key: k, Value: v})
	}
	if config.Sampler != nil {
		jconfig.Sampler = &jaegerconfig.SamplerConfig{
			Type:                    config.Sampler.Type,
			Param:                   config.Sampler.Param,
			SamplingServerURL:       config.Sampler.SamplingServerURL,
			MaxOperations:           config.Sampler.MaxOperations,
			SamplingRefreshInterval: config.Sampler.SamplingRefreshInterval,
		}
	}
	if config.Reporter != nil {
		jconfig.Reporter = &jaegerconfig.ReporterConfig{
			QueueSize:           config.Reporter.QueueSize,
			BufferFlushInterval: config.Reporter.BufferFlushInterval,
			LogSpans:            config.Reporter.LogSpans,
			LocalAgentHostPort:  config.Reporter.LocalAgentHostPort,
			CollectorEndpoint:   config.Reporter.CollectorEndpoint,
			User:                config.Reporter.User,
			Password:            config.Reporter.Password,
		}
	}
	if config.Headers != nil {
		jconfig.Headers = &jaeger.HeadersConfig{
			JaegerDebugHeader:        config.Headers.JaegerDebugHeader,
			JaegerBaggageHeader:      config.Headers.JaegerBaggageHeader,
			TraceContextHeaderName:   config.Headers.TraceContextHeaderName,
			TraceBaggageHeaderPrefix: config.Headers.TraceBaggageHeaderPrefix,
		}
	}
	if config.BaggageRestrictions != nil {
		jconfig.BaggageRestrictions = &jaegerconfig.BaggageRestrictionsConfig{
			DenyBaggageOnInitializationFailure: config.BaggageRestrictions.DenyBaggageOnInitializationFailure,
			HostPort:                           config.BaggageRestrictions.HostPort,
			RefreshInterval:                    config.BaggageRestrictions.RefreshInterval,
		}
	}
	if config.Throttler != nil {
		jconfig.Throttler = &jaegerconfig.ThrottlerConfig{
			HostPort:                  config.Throttler.HostPort,
			RefreshInterval:           config.Throttler.RefreshInterval,
			SynchronousInitialization: config.Throttler.SynchronousInitialization,
		}
	}

	opts := tracing.TracerOptions{
		Logger: log.NopLogger(),
		Stats:  stats.NopStats(),
	}
	for _, o := range o {
		o(&opts)
	}

	jmet := jaegermetrics.NullFactory // TODO: Wrap spine stats

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := jconfig.NewTracer(
		jaegerconfig.Logger(&Logger{L: opts.Logger}),
		jaegerconfig.Metrics(jmet),
	)
	if err != nil {
		return nil, err
	}

	return &Tracer{
		Tracer: tracer,
		Closer: closer,
	}, nil
}

type Tracer struct {
	// Tracer is a jaeger Tracer instance
	Tracer opentracing.Tracer
	// Closer is the jaeger Closer that can be used to flush buffers before shutdown
	Closer io.Closer
}

func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return t.Tracer.StartSpan(operationName, opts...)
}

func (t *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return t.Tracer.Inject(sm, format, carrier)
}

func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return t.Tracer.Extract(format, carrier)
}

func (t *Tracer) Close() error {
	return t.Closer.Close()
}

// Logger wraps Jaeger logs with spine
type Logger struct {
	// L is a spine logger
	L log.Logger
}

// Error logs a message at error priority
func (l *Logger) Error(msg string) {
	// Log it as a warning in case Jaeger takes the error level lightly
	l.L.Warning("tracing.jaeger.err", msg)
}

// Infof logs a message at info priority
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.L.Trace("tracing.jaeger.info", fmt.Sprintf(msg, args...))
}

// Metrics wrap Jaeger metrics with spine
// TODO:
// type Metrics struct{}
//
// func (m *Metrics) Counter(name string, tags map[string]string) metrics.Counter {}
// func (m *Metrics) Timer(name string, tags map[string]string) metrics.Timer     {}
// func (m *Metrics) Gauge(name string, tags map[string]string) metrics.Gauge     {}
//
// // Namespace returns a nested metrics factory.
// func (m *Metrics) Namespace(name string, tags map[string]string) metrics.Factory {}
