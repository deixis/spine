// Package file prints log lines to a file.
package file

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/pkg/errors"
)

const (
	Name = "file"

	defaultMode        = 0660
	flag               = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	newLine            = '\n'
	defaultFlushPeriod = 5 * time.Second
)

// Config defines the filer printer config
type Config struct {
	Path        string `toml:"path"`
	Flag        int    `toml:"flag"`
	Mode        uint32 `toml:"mode"`
	FlushPeriod int    `toml:"flush_period"`
}

func New(tree config.Tree) (log.Printer, error) {
	c := Config{}
	if err := tree.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Path == "" {
		return nil, errors.New("missing \"path\" on file log printer config")
	}
	if c.Mode == 0 {
		c.Mode = defaultMode
	}
	flushPeriod := defaultFlushPeriod
	if c.FlushPeriod > 0 {
		flushPeriod = time.Duration(c.FlushPeriod) * time.Second
	}

	l := &Logger{
		conf:    c,
		sighup:  make(chan os.Signal, 1),
		flusher: make(chan struct{}, 1),
	}
	go l.listen()
	go l.flushPeriodically(flushPeriod)
	return l, l.open()
}

type Logger struct {
	mu sync.Mutex

	conf    Config
	buf     *bufio.Writer
	file    *os.File
	sighup  chan os.Signal
	flusher chan struct{}
}

func (l *Logger) Print(ctx *log.Context, s string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.buf.WriteString(s)
	if err != nil {
		return err
	}
	return l.buf.WriteByte(newLine)
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	signal.Stop(l.sighup)
	close(l.sighup)

	l.flusher <- struct{}{}

	return l.close()
}

func (l *Logger) open() (err error) {
	l.file, err = os.OpenFile(l.conf.Path, flag, os.FileMode(l.conf.Mode))
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	l.buf = bufio.NewWriter(l.file)
	return nil
}

func (l *Logger) close() error {
	l.buf.Flush()
	return l.file.Close()
}

// listen listens to SIGHUP signals to reopen the log file.
// Logrotated can be configured to send a SIGHUP signal to a process after
// rotating it's logs.
func (l *Logger) listen() {
	signal.Notify(l.sighup, syscall.SIGHUP)
	for range l.sighup {
		l.mu.Lock()
		defer l.mu.Unlock()

		fmt.Fprintf(os.Stderr, "%s: Reopening %q\n", time.Now(), l.conf.Path)
		if err := l.close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: Error closing log file: %s\n", time.Now(), err)
		}
		if err := l.open(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: Error opening log file: %s\n", time.Now(), err)
		}
	}
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
				if err := l.buf.Flush(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: Error flushing buffer (%s)\n", time.Now(), err)
				}
			}()
		}
	}
}
