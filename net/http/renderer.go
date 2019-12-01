package http

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Renderer is a response returned by an action
type Renderer interface {
	// Render writes a response to the response writer
	Render(ResponseWriter) error
}

// RenderJSON is a renderer that marshals responses in JSON
type RenderJSON struct {
	Code int
	V    interface{}
}

func (r *RenderJSON) Render(res ResponseWriter) error {
	// Header
	res.Header().Add("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(r.Code)

	// Body
	if err := json.NewEncoder(res).Encode(r.V); err != nil {
		return err
	}
	return nil
}

// RenderGob is a renderer that marshals responses in Gob
type RenderGob struct {
	Code int
	V    interface{}
}

func (r *RenderGob) Render(res ResponseWriter) error {
	// Header
	res.Header().Add("Content-Type", "application/octet-stream")
	res.WriteHeader(r.Code)

	// Body
	if err := gob.NewEncoder(res).Encode(r.V); err != nil {
		return err
	}
	return nil
}

// RenderHead is a renderer that returns a body-less response
type RenderHead struct {
	Code int
}

func (r *RenderHead) Render(res ResponseWriter) error {
	res.WriteHeader(r.Code)
	return nil
}

// RenderRedirect is a renderer that returns a redirection
type RenderRedirect struct {
	Req *http.Request
	URL string
}

func (r *RenderRedirect) Render(res ResponseWriter) error {
	http.Redirect(res, r.Req, r.URL, http.StatusTemporaryRedirect)
	return nil
}

type RenderData struct {
	Code        int
	ContentType string
	Reader      io.ReadCloser
}

func (r *RenderData) Render(res ResponseWriter) error {
	defer r.Reader.Close()
	if r.ContentType != "" {
		res.Header()["Content-Type"] = []string{r.ContentType}
	}
	res.WriteHeader(r.Code)
	_, err := io.Copy(res, r.Reader)
	return err
}

type RenderContent struct {
	Req     *http.Request
	Modtime time.Time
	Content io.ReadSeeker
}

func (r *RenderContent) Render(res ResponseWriter) error {
	http.ServeContent(res, r.Req, "", r.Modtime, r.Content)
	return nil
}

type RenderConditional struct {
	Req          *http.Request
	ETag         string
	LastModified time.Time
	Next         Renderer
}

func (r *RenderConditional) Render(res ResponseWriter) error {
	lastModified := r.LastModified.UTC().Format(http.TimeFormat)
	if r.ETag != "" {
		res.Header().Add("ETag", r.ETag)
	}
	if !r.LastModified.IsZero() {
		res.Header().Add("Last-Modified", lastModified)
	}

	if (r.ETag != "" && r.Req.Header.Get("If-None-Match") == r.ETag) ||
		(lastModified != "" && r.Req.Header.Get("If-Modified-Since") == lastModified) {
		res.WriteHeader(http.StatusNotModified)
		return nil
	}
	return r.Next.Render(res)
}
