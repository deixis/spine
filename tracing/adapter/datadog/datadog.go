package datadog

import (
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const Name = "datadog"

func New(tree config.Tree, o ...tracing.TracerOption) (tracing.Tracer, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal tracing.jaeger config")
	}

	opts := tracing.TracerOptions{
		Logger: log.NopLogger(),
		Stats:  stats.NopStats(),
	}
	for _, o := range o {
		o(&opts)
	}

	so := []tracer.StartOption{
		tracer.WithService(config.Service.Name),
		tracer.WithServiceVersion(config.Service.Version),
		tracer.WithLogger(&Logger{opts.Logger}),
		tracer.WithAnalytics(config.Analytics.Enabled),
	}
	if config.AgentAddr != "" {
		so = append(so, tracer.WithAgentAddr(config.AgentAddr))
	}
	if config.Analytics.Rate != nil {
		so = append(so, tracer.WithAnalyticsRate(*config.Analytics.Rate))
	}
	for k, v := range config.Tags {
		so = append(so, tracer.WithGlobalTag(k, v))
	}

	// Start a Datadog tracer
	return &Tracer{Tracer: opentracer.New(so...)}, nil
}

type Tracer struct {
	Tracer opentracing.Tracer
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
	// Globally flush any buffered traces before shutting down
	// since there is no instance-specific closer
	tracer.Flush()
	return nil
}

// Logger wraps Datadog logs with spine
type Logger struct {
	// L is a spine logger
	L log.Logger
}

// Log prints the given message.
func (l *Logger) Log(msg string) {
	l.L.Trace("tracing.datadog", msg)
}
