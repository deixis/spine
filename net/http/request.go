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

// Path returns the matched HTTP endpoint path
// e.g. /users/{id} and not /users/123
func (r *Request) Path() string {
	return r.path
}

// Method returns the matched HTTP endpoint method
// e.g. GET, POST, PUT, DELETE
func (r *Request) Method() string {
	return r.method
}
