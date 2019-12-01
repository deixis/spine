package http

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"mime"

	"github.com/deixis/spine/log"
)

const (
	mimeJSON = "application/json"
	// mimeGob is a custom MIME type for Gob. application/octet-stream
	// should probably be favoured over this unofficial type, but this one
	// allows us to dynamically select the Gob parser
	mimeGob = "application/x-gob"
)

// Parser is a request data Parser
type Parser interface {
	Type() string
	Parse(v interface{}) error
}

// pickParser selects a Parser for the request content-type
func pickParser(ctx context.Context, req *Request) Parser {
	log.Trace(ctx, "http.parser.content_length", "Request content length",
		log.Int64("len", req.HTTP.ContentLength),
	)

	// If content type is not provided and the request body is empty,
	// then there is no need to pick a parser
	ct := req.HTTP.Header.Get("Content-Type")
	if ct == "" {
		log.Trace(ctx, "http.parser.no_content_type", "Pick null parser")
		return &ParseNull{"", req.HTTP.ContentLength}
	}

	// Parse mime type
	m, _, err := mime.ParseMediaType(ct)
	if err != nil {
		log.Warn(ctx, "http.content_type.err", "Cannot parse Content-Type",
			log.String("content_type", ct),
			log.Error(err),
		)
	}

	switch m {
	case mimeJSON:
		log.Trace(ctx, "http.parser", "Pick JSON parser", log.String("type", mimeJSON))
		return &ParseJSON{req}
	case mimeGob:
		log.Trace(ctx, "http.parser", "Pick Gob parser", log.String("type", mimeGob))
		return &ParseGob{req}
	}

	log.Trace(ctx, "http.parser", "Pick null parser", log.String("type", "null"))
	return &ParseNull{m, req.HTTP.ContentLength}
}

// ParseJSON Parses JSON
type ParseJSON struct {
	req *Request
}

// Type returns the mime type
func (d *ParseJSON) Type() string {
	return mimeJSON
}

// Parse unmarshal the request payload into the given structure
func (d *ParseJSON) Parse(v interface{}) error {
	if err := json.NewDecoder(d.req.HTTP.Body).Decode(v); err != nil {
		return err
	}
	defer d.req.HTTP.Body.Close()
	return nil
}

// ParseGob Parses Gob
type ParseGob struct {
	req *Request
}

// Type returns the mime type
func (d *ParseGob) Type() string {
	return mimeGob
}

// Parse unmarshal the request payload into the given structure
func (d *ParseGob) Parse(v interface{}) error {
	if err := gob.NewDecoder(d.req.HTTP.Body).Decode(v); err != nil {
		return err
	}
	defer d.req.HTTP.Body.Close()
	return nil
}

// ParseNull is a null-object that is used when no other Parsers have been found
type ParseNull struct {
	mime   string
	length int64
}

// Type returns the mime type
func (d *ParseNull) Type() string {
	return ""
}

// Parse returns an error because this Parser has nothing to Parse
func (d *ParseNull) Parse(v interface{}) error {
	return fmt.Errorf("ParseNull: %s (%d)", d.mime, d.length)
}
