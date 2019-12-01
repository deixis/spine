package http

import (
	"crypto/tls"
	"time"
)

// Option allows to configure unexported handler fields
type Option func(*Server)

// OptTLS changes the handler TLS configuration.
func OptTLS(config *tls.Config) Option {
	return func(s *Server) {
		s.http.TLSConfig = config
	}
}

// OptReadTimeout configures the maximum duration for reading the entire
// request, including the body.
func OptReadTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.http.ReadHeaderTimeout = d
	}
}

// OptReadHeaderTimeout configures the amount of time allowed to read
// request headers
func OptReadHeaderTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.http.ReadHeaderTimeout = d
	}
}

// OptWriteTimeout configures the maximum duration before timing out
// writes of the response
func OptWriteTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.http.WriteTimeout = d
	}
}

// OptIdleTimeout configures the maximum amount of time to wait for the
// next request when keep-alives are enabled.
func OptIdleTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.http.IdleTimeout = d
	}
}
