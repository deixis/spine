// Package json is a JSON log formatter.
//
// It is a good solution for production environment where log lines
// are usually sent to a log aggregator, such as Elasticsearch (ELK stack), or Splunk.
package json

import (
	"encoding/json"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
)

const Name = "json"

func New(c config.Tree) (log.Formatter, error) {
	return &Formatter{}, nil
}

type Formatter struct{}

func (f *Formatter) Format(ctx *log.Ctx, tag, msg string, fields ...log.Field) (string, error) {
	out := &out{
		Level:     ctx.Level,
		Timestamp: ctx.Timestamp,
		Service:   ctx.Service,
		File:      ctx.File,
		Tag:       tag,
		Msg:       msg,
		Fields:    formatFields(fields),
	}

	r, err := json.Marshal(out)
	if err != nil {
		return "", err
	}

	return string(r), nil
}

func formatFields(fields []log.Field) map[string]interface{} {
	m := map[string]interface{}{}
	for _, f := range fields {
		k, v := f.KV()
		m[k] = v
	}
	return m
}

type out struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Service   string                 `json:"service"`
	File      string                 `json:"file"`
	Tag       string                 `json:"tag,omitempty"`
	Msg       string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields"`
}
