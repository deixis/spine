// Package stackdriver send log lines to Google Stackdriver.
//
// It is based on https://godoc.org/cloud.google.com/go/logging
package stackdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
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
	// CommonLabels are labels that apply to all log entries written from a Logger,
	// so that you don't have to repeat them in each log entry's Labels field. If
	// any of the log entries contains a (key, value) with the same key that is in
	// CommonLabels, then the entry's (key, value) overrides the one in
	// CommonLabels.
	CommonLabels map[string]string `toml:"common_labels"`
	// Credentials allows to define authentication credentials from the config
	// file instead of the GOOGLE_APPLICATION_CREDENTIALS environment variable.
	Credentials ConfigCredentials `toml:"credentials"`
}

type ConfigCredentials struct {
	Type                    string `json:"type" toml:"type"`
	ProjectID               string `json:"project_id" toml:"project_id"`
	PrivateKeyID            string `json:"private_key_id" toml:"private_key_id"`
	PrivateKey              string `json:"private_key" toml:"private_key"`
	ClientEmail             string `json:"client_email" toml:"client_email"`
	ClientID                string `json:"client_id" toml:"client_id"`
	AuthURI                 string `json:"auth_uri" toml:"auth_uri"`
	TokenURI                string `json:"token_uri" toml:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url" toml:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url" toml:"client_x509_cert_url"`
}

func (c *ConfigCredentials) marshalJSON() ([]byte, error) {
	// Set default values
	if c.Type == "" {
		c.Type = "service_account"
	}
	if c.AuthURI == "" {
		c.AuthURI = "https://accounts.google.com/o/oauth2/auth"
	}
	if c.TokenURI == "" {
		c.TokenURI = "https://oauth2.googleapis.com/token"
	}
	if c.AuthProviderX509CertURL == "" {
		c.AuthProviderX509CertURL = "https://www.googleapis.com/oauth2/v1/certs"
	}

	// Private keys loaded can be escape sometimes
	c.PrivateKey = strings.ReplaceAll(c.PrivateKey, "\\n", "\n")

	json, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling Google credentials")
	}
	return json, nil
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

	var options []option.ClientOption

	// Auth from config file
	if c.Credentials.ProjectID != "" && c.Credentials.PrivateKey != "" {
		creds, err := c.Credentials.marshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "error in \"credentials\" on stackdriver log printer config")
		}
		options = append(options, option.WithCredentialsJSON(creds))
	}

	// Create a Client
	ctx := context.Background()
	client, err := logging.NewClient(
		ctx,
		c.Parent,
		options...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failing to initialise Stackdriver client")
	}

	var opts []logging.LoggerOption
	if c.CommonLabels != nil {
		opts = append(opts, logging.CommonLabels(c.CommonLabels))
	}

	// Test connection to Stackdriver
	if err := client.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "failing to ping Stackdriver")
	}

	l := &Logger{
		flusher: make(chan struct{}, 1),
		C:       client,
		L:       client.Logger(c.LogID, opts...),
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
