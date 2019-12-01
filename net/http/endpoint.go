package http

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

// ServeFunc is the function signature for standard endpoints
type ServeFunc func(ctx context.Context, w ResponseWriter, r *Request)

// An Endpoint is a an entity that serves a request from a given route
type Endpoint interface {
	Path() string
	Method() string
	Attach(*mux.Router, func(http.ResponseWriter, *http.Request))
	Serve(ctx context.Context, w ResponseWriter, r *Request)
}

type stdEndpoint struct {
	method     string
	path       string
	handleFunc func(ctx context.Context, w ResponseWriter, r *Request)
}

func (h *stdEndpoint) Path() string {
	return h.path
}

func (h *stdEndpoint) Method() string {
	return h.method
}

func (h *stdEndpoint) Attach(r *mux.Router, f func(http.ResponseWriter,
	*http.Request)) {
	r.HandleFunc(h.path, f).Methods(h.method, OPTIONS)
}

func (h *stdEndpoint) Serve(ctx context.Context, w ResponseWriter, r *Request) {
	h.handleFunc(ctx, w, r)
}

type fileEndpoint struct {
	path        string
	fileHandler *fileHandler
	hook        func(ctx context.Context, w ResponseWriter, r *Request, serve func())
}

func (h *fileEndpoint) Path() string {
	return h.path
}

func (h *fileEndpoint) Method() string {
	return GET
}

func (h *fileEndpoint) Attach(r *mux.Router, f func(http.ResponseWriter,
	*http.Request)) {
	r.PathPrefix(h.path).Handler(http.StripPrefix(h.path, h.fileHandler))
}

func (h *fileEndpoint) Serve(ctx context.Context, w ResponseWriter, r *Request) {
	serveFile := func() { h.fileHandler.ServeHTTP(w, r.HTTP) }
	if h.hook != nil {
		h.hook(ctx, w, r, serveFile)
		return
	}
	serveFile()
}
