package http

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DefaultClient is the default Client and is used by Get, Head, and Post.
var DefaultClient = &Client{}

// Client is a wrapper for the standard net/http client.
type Client struct {
	// HTTP is the standard net/http client
	HTTP http.Client
	// PropagateContext tells whether the context should be propagated upstream
	//
	// This should be activated when the upstream endpoint is a SPINE service
	// or another SPINE-compatible service. The context can potentially leak
	// sensitive information, so do not activate it for services that you don't trust.
	PropagateContext bool
}

// Do sends an HTTP request with the provided http.Client and returns
// an HTTP response.
//
// If the client is nil, http.DefaultClient is used.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.PropagateContext {
		// TODO: Propagate transit
		// TODO: Propagate opentracing span
		err := encodeContext(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := c.HTTP.Do(req.WithContext(ctx))
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	return resp, nil
}

// Get issues a GET request via the Do function.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(GET, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Head issues a HEAD request via the Do function.
func (c *Client) Head(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(HEAD, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post issues a POST request via the Do function.
func (c *Client) Post(
	ctx context.Context, url string, bodyType string, body io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequest(POST, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(ctx, req)
}

// PostForm issues a POST request via the Do function.
func (c *Client) PostForm(
	ctx context.Context, url string, data url.Values,
) (*http.Response, error) {
	return c.Post(ctx, url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Get issues a GET request via the Do function.
func Get(ctx context.Context, url string) (*http.Response, error) {
	return DefaultClient.Get(ctx, url)
}

// Post issues a POST to the specified URL.
func Post(
	ctx context.Context, url string, contentType string, body io.Reader,
) (resp *http.Response, err error) {
	return DefaultClient.Post(ctx, url, contentType, body)
}

// PostForm issues a POST to the specified URL, with data's keys and
// values URL-encoded as the request body.
func PostForm(
	ctx context.Context, url string, data url.Values,
) (resp *http.Response, err error) {
	return DefaultClient.PostForm(ctx, url, data)
}

// Head issues a HEAD to the specified URL. If the response is one of
// the following redirect codes, Head follows the redirect, up to a
// maximum of 10 redirects:
//
//    301 (Moved Permanently)
//    302 (Found)
//    303 (See Other)
//    307 (Temporary Redirect)
//    308 (Permanent Redirect)
//
// Head is a wrapper around DefaultClient.Head
func Head(ctx context.Context, url string) (resp *http.Response, err error) {
	return DefaultClient.Head(ctx, url)
}
