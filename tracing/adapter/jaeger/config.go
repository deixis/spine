package jaeger

import (
	"time"
)

// Config configures Jaeger Tracer
type Config struct {
	// ServiceName specifies the service name to use on the tracer.
	// Can be provided via environment variable named JAEGER_SERVICE_NAME
	ServiceName string `toml:"service_name"`

	// Disabled can be provided via environment variable named JAEGER_DISABLED
	Disabled bool `toml:"disabled"`

	// RPCMetrics can be provided via environment variable named JAEGER_RPC_METRICS
	RPCMetrics bool `toml:"rpc_metrics"`

	// Tags can be provided via environment variable named JAEGER_TAGS
	Tags map[string]string `toml:"tags"`

	Sampler             *SamplerConfig             `toml:"sampler"`
	Reporter            *ReporterConfig            `toml:"reporter"`
	Headers             *HeadersConfig             `toml:"headers"`
	BaggageRestrictions *BaggageRestrictionsConfig `toml:"baggage_restrictions"`
	Throttler           *ThrottlerConfig           `toml:"throttler"`
}

// SamplerConfig allows initializing a non-default sampler.  All fields are optional.
type SamplerConfig struct {
	// Type specifies the type of the sampler: const, probabilistic, rateLimiting, or remote
	// Can be set by exporting an environment variable named JAEGER_SAMPLER_TYPE
	Type string `toml:"type"`

	// Param is a value passed to the sampler.
	// Valid values for Param field are:
	// - for "const" sampler, 0 or 1 for always false/true respectively
	// - for "probabilistic" sampler, a probability between 0 and 1
	// - for "rateLimiting" sampler, the number of spans per second
	// - for "remote" sampler, param is the same as for "probabilistic"
	//   and indicates the initial sampling rate before the actual one
	//   is received from the mothership.
	// Can be set by exporting an environment variable named JAEGER_SAMPLER_PARAM
	Param float64 `toml:"param"`

	// SamplingServerURL is the address of jaeger-agent's HTTP sampling server
	// Can be set by exporting an environment variable named JAEGER_SAMPLER_MANAGER_HOST_PORT
	SamplingServerURL string `toml:"sampling_server_url"`

	// MaxOperations is the maximum number of operations that the sampler
	// will keep track of. If an operation is not tracked, a default probabilistic
	// sampler will be used rather than the per operation specific sampler.
	// Can be set by exporting an environment variable named JAEGER_SAMPLER_MAX_OPERATIONS
	MaxOperations int `toml:"max_operations"`

	// SamplingRefreshInterval controls how often the remotely controlled sampler will poll
	// jaeger-agent for the appropriate sampling strategy.
	// Can be set by exporting an environment variable named JAEGER_SAMPLER_REFRESH_INTERVAL
	SamplingRefreshInterval time.Duration `toml:"sampling_refresh_interval"`
}

// ReporterConfig configures the reporter. All fields are optional.
type ReporterConfig struct {
	// QueueSize controls how many spans the reporter can keep in memory before it starts dropping
	// new spans. The queue is continuously drained by a background go-routine, as fast as spans
	// can be sent out of process.
	// Can be set by exporting an environment variable named JAEGER_REPORTER_MAX_QUEUE_SIZE
	QueueSize int `toml:"queue_size"`

	// BufferFlushInterval controls how often the buffer is force-flushed, even if it's not full.
	// It is generally not useful, as it only matters for very low traffic services.
	// Can be set by exporting an environment variable named JAEGER_REPORTER_FLUSH_INTERVAL
	BufferFlushInterval time.Duration

	// LogSpans, when true, enables LoggingReporter that runs in parallel with the main reporter
	// and logs all submitted spans. Main Configuration.Logger must be initialized in the code
	// for this option to have any effect.
	// Can be set by exporting an environment variable named JAEGER_REPORTER_LOG_SPANS
	LogSpans bool `toml:"log_spans"`

	// LocalAgentHostPort instructs reporter to send spans to jaeger-agent at this address
	// Can be set by exporting an environment variable named JAEGER_AGENT_HOST / JAEGER_AGENT_PORT
	LocalAgentHostPort string `toml:"local_agent_host_port"`

	// CollectorEndpoint instructs reporter to send spans to jaeger-collector at this URL
	// Can be set by exporting an environment variable named JAEGER_ENDPOINT
	CollectorEndpoint string `toml:"collector_endpoint"`

	// User instructs reporter to include a user for basic http authentication when sending spans to jaeger-collector.
	// Can be set by exporting an environment variable named JAEGER_USER
	User string `toml:"user"`

	// Password instructs reporter to include a password for basic http authentication when sending spans to
	// jaeger-collector. Can be set by exporting an environment variable named JAEGER_PASSWORD
	Password string `toml:"password"`
}

// HeadersConfig contains the values for the header keys that Jaeger will use.
// These values may be either custom or default depending on whether custom
// values were provided via a configuration.
type HeadersConfig struct {
	// JaegerDebugHeader is the name of HTTP header or a TextMap carrier key which,
	// if found in the carrier, forces the trace to be sampled as "debug" trace.
	// The value of the header is recorded as the tag on the root span, so that the
	// trace can be found in the UI using this value as a correlation ID.
	JaegerDebugHeader string `yaml:"jaegerDebugHeader"`

	// JaegerBaggageHeader is the name of the HTTP header that is used to submit baggage.
	// It differs from TraceBaggageHeaderPrefix in that it can be used only in cases where
	// a root span does not exist.
	JaegerBaggageHeader string `yaml:"jaegerBaggageHeader"`

	// TraceContextHeaderName is the http header name used to propagate tracing context.
	// This must be in lower-case to avoid mismatches when decoding incoming headers.
	TraceContextHeaderName string `yaml:"TraceContextHeaderName"`

	// TraceBaggageHeaderPrefix is the prefix for http headers used to propagate baggage.
	// This must be in lower-case to avoid mismatches when decoding incoming headers.
	TraceBaggageHeaderPrefix string `yaml:"traceBaggageHeaderPrefix"`
}

// BaggageRestrictionsConfig configures the baggage restrictions manager which can be used to whitelist
// certain baggage keys. All fields are optional.
type BaggageRestrictionsConfig struct {
	// DenyBaggageOnInitializationFailure controls the startup failure mode of the baggage restriction
	// manager. If true, the manager will not allow any baggage to be written until baggage restrictions have
	// been retrieved from jaeger-agent. If false, the manager wil allow any baggage to be written until baggage
	// restrictions have been retrieved from jaeger-agent.
	DenyBaggageOnInitializationFailure bool `toml:"deny_baggage_on_initialization_failure"`

	// HostPort is the hostPort of jaeger-agent's baggage restrictions server
	HostPort string `toml:"host_port"`

	// RefreshInterval controls how often the baggage restriction manager will poll
	// jaeger-agent for the most recent baggage restrictions.
	RefreshInterval time.Duration `toml:"refresh_interval"`
}

// ThrottlerConfig configures the throttler which can be used to throttle the
// rate at which the client may send debug requests.
type ThrottlerConfig struct {
	// HostPort of jaeger-agent's credit server.
	HostPort string `toml:"host_port"`

	// RefreshInterval controls how often the throttler will poll jaeger-agent
	// for more throttling credits.
	RefreshInterval time.Duration `toml:"refresh_interval"`

	// SynchronousInitialization determines whether or not the throttler should
	// synchronously fetch credits from the agent when an operation is seen for
	// the first time. This should be set to true if the client will be used by
	// a short lived service that needs to ensure that credits are fetched
	// upfront such that sampling or throttling occurs.
	SynchronousInitialization bool `toml:"synchronous_initialization"`
}
