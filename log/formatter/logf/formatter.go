// Package logf is a human friendly log formatter.
//
// It is ideal for a development environment where
// log lines are almost exlusively consumed by developers
package logf

import (
	"fmt"
	"strings"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
)

const Name = "logf"

func New(c config.Tree) (log.Formatter, error) {
	return &Formatter{}, nil
}

type Formatter struct{}

func (f *Formatter) Format(ctx *log.Ctx, tag, msg string, fields ...log.Field) (string, error) {
	base := strings.Join([]string{
		ctx.Level,
		ctx.Timestamp,
		ctx.Service,
		ctx.File,
	}, " ")
	// Add padding
	padding := 75 - len(base)
	if padding > 0 {
		base = base + strings.Repeat(" ", padding)
	}

	l := []string{}
	if msg != "" {
		l = append(l, msg)
	}
	l = append(l, formatFields(fields)...)
	content := strings.Join(l, " ")

	if tag != "" {
		return fmt.Sprintf("%s [%s] %s", base, tag, content), nil
	}
	return fmt.Sprintf("%s %s", base, content), nil
}

func formatFields(fields []log.Field) []string {
	l := make([]string, len(fields))
	for i, f := range fields {
		k, v := f.KV()
		l[i] = fmt.Sprintf("<%s=%s>", k, v)
	}
	return l
}

type out struct {
	Level     string
	Timestamp string
	ID        string
	AppType   string
	AppRev    string
	Hostname  string
	File      string
	Type      string
	Tag       string
	Msg       string
	Fields    []kv
}

type kv struct {
	K string
	V string
}
