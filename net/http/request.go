package http

import (
	"context"
	"net/http"
	"time"
)

// Request wraps the standard net/http Request struct
type Request struct {
	startTime time.Time
	method    string
	path      string

	HTTP   *http.Request
	Params map[string]string
}

// Parse parses the request body and decodes it on the given struct
func (r *Request) Parse(ctx context.Context, v interface{}) error {
	return pickParser(ctx, r).Parse(v)
}
