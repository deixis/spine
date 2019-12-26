// Package stackdriver send log lines to Google Stackdriver.
//
// It is based on https://godoc.org/cloud.google.com/go/logging
package stackdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/pkg/errors"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

const (
	Name = "stackdriver"

	defaultFlushPeriod = 5 * time.Second
)

// Config defines the filer printer config
type Config struct {
	// A Parent can take any of the following forms:
	//
	// - projects/PROJECT_ID
	// - folders/FOLDER_ID
	// - billingAccounts/ACCOUNT_ID
	// - organizations/ORG_ID
	//
	// for backwards compatibility, a string with no '/' is also allowed and is
	// interpreted as a project ID.
	// ProjectID sets the Google Cloud Platform project ID.
	Parent string `toml:"parent"`
	// Name sets the name of the log to write to.
	//
	// A log ID must be less than 512 characters long and can only
	// include the following characters: upper and lower case alphanumeric
	// characters: [A-Za-z0-9]; and punctuation characters: forward-slash,
	// underscore, hyphen, and period.
	LogID string `toml:"log_id"`
	// FlushPeriod is the frequence on which log lines are flushed to StackDriver
	FlushPeriod int `toml:"flush_period"`
}

func New(tree config.Tree) (log.Printer, error) {
	c := Config{}
	if err := tree.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Parent == "" {
		return nil, errors.New("missing \"Parent\" on stackdriver log printer config")
	}
	if c.LogID == "" {
		return nil, errors.New("missing \"LogID\" on stackdriver log printer config")
	}
	flushPeriod := defaultFlushPeriod
	if c.FlushPeriod > 0 {
		flushPeriod = time.Duration(c.FlushPeriod) * time.Second
	}

	// Create a Client
	ctx := context.Background()
	client, err := logging.NewClient(ctx, c.Parent)
	if err != nil {
		return nil, errors.Wrap(err, "failing to initialise Stackdriver client")
	}

	// Test connection to Stackdriver
	if err := client.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "failing to ping Stackdriver")
	}

	l := &Logger{
		flusher: make(chan struct{}, 1),
		C:       client,
		L:       client.Logger(c.LogID),
	}
	go l.flushPeriodically(flushPeriod)
	return l, nil
}

type Logger struct {
	mu      sync.Mutex
	flusher chan struct{}

	C *logging.Client
	L *logging.Logger
}

func (l *Logger) Print(ctx *log.Context, s string) error {
	entry := logging.Entry{
		Timestamp: ctx.Timestamp,
		Payload:   json.RawMessage([]byte(s)), // Assuming JSON formatter
		Labels: map[string]string{
			"service": ctx.Service,
		},
		SourceLocation: &logpb.LogEntrySourceLocation{
			// Source file name. Depending on the runtime environment, this
			// might be a simple name or a fully-qualified name.
			File: ctx.File,
			// Line within the source file. 1-based; 0 indicates no line number
			// available.
			Line: ctx.Line,
		},
	}

	// Translate internal log level to Stackdriver level
	switch ctx.Level {
	case log.LevelTrace:
		// Debug means debug or trace information.
		entry.Severity = logging.Debug
	case log.LevelWarning:
		// Warning means events that might cause problems.
		entry.Severity = logging.Warning
	case log.LevelError:
		// Alert means a person must take an action immediately.
		entry.Severity = logging.Alert
	default:
		entry.Severity = logging.Default
	}

	l.L.Log(entry)
	return nil
}

func (l *Logger) Close() error {
	l.flusher <- struct{}{}
	return l.C.Close() // Flush and exit
}

func (l *Logger) flushPeriodically(d time.Duration) {
	tick := time.Tick(d)
	for {
		select {
		case <-l.flusher:
			return
		case <-tick:
			func() {
				l.mu.Lock()
				defer l.mu.Unlock()
				if err := l.L.Flush(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: Error flushing Stackdriver buffer (%s)\n", time.Now(), err)
				}
			}()
		}
	}
}
