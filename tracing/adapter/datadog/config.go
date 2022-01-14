package datadog

// Config configures DataDog Tracer
type Config struct {
	// AgentAddr sets address where the agent is located. The default is
	// localhost:8126. It should contain both host and port.
	AgentAddr string `toml:"agent_addr"`
	// Service-related configuration, such as name and version
	Service ServiceConfig `toml:"service"`
	// DataDog Analytics-related configuration
	Analytics AnalyticsConfig `toml:"analytics"`
	// Tags sets a key/value pair which will be set as a tag on all spans
	// created by tracer.
	Tags map[string]string `toml:"tags"`
}

type ServiceConfig struct {
	// Name specifies the service name to use on the tracer.
	Name string `toml:"name"`
	// Version specifies the version of the service that is running. This will
	// be included in spans from this service in the "version" tag.
	Version string `toml:"version"`
}

type AnalyticsConfig struct {
	// Enabled tells whether Trace Search & Analytics should be enabled for integrations.
	Enabled bool `toml:"enabled"`
	// Rate sets the global sampling rate for sampling APM events.
	Rate *float64 `toml:"rate"`
}
