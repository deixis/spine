package logger

import (
	"fmt"
	"runtime"
	"time"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/log/formatter"
	"github.com/deixis/spine/log/printer"
)

// New creates a new logger
func New(service string, config config.Tree) (log.Logger, error) {
	lc := &Config{}
	config.Unmarshal(lc)

	f, err := formatter.New(config.Get("formatter"))
	if err != nil {
		return nil, err
	}

	p, err := printer.New(config.Get("printer"))
	if err != nil {
		return nil, err
	}

	return &Logger{
		service:   service,
		level:     log.ParseLevel(lc.Level),
		fmt:       f,
		pnt:       p,
		calldepth: 1,
	}, nil
}

// Logger is the key struct of the log package.
// It is the part that links the log formatter to the log printer
type Logger struct {
	service   string
	level     log.Level
	fmt       log.Formatter
	pnt       log.Printer
	calldepth int

	fields []log.Field
}

// Trace creates a trace log line.
// Trace level logs are to follow the code executio step by step
func (l *Logger) Trace(tag, msg string, fields ...log.Field) {
	l.log(log.LevelTrace, tag, msg, fields...)
}

// Warning creates a trace log line.
// Warning level logs are meant to draw attention above a certain threshold
func (l *Logger) Warning(tag, msg string, fields ...log.Field) {
	l.log(log.LevelWarning, tag, msg, fields...)
}

// Error creates a trace log line.
// Error level logs need immediate attention
// The 2AM rule applies here, which means that if you are on call, this log line will wake you up at 2AM
func (l *Logger) Error(tag, msg string, fields ...log.Field) {
	l.log(log.LevelError, tag, msg, fields...)
}

// With adds the given fields to a cloned logger
func (l *Logger) With(fields ...log.Field) log.Logger {
	c := l.clone()
	c.fields = append(c.fields, fields...)
	return c
}

// AddCalldepth clones the logger and changes the call depth
func (l *Logger) AddCalldepth(n int) log.Logger {
	c := l.clone()
	c.calldepth = c.calldepth + n
	return c
}

func (l *Logger) Close() error {
	return l.pnt.Close()
}

func (l *Logger) clone() *Logger {
	return &Logger{
		service:   l.service,
		level:     l.level,
		fmt:       l.fmt,
		pnt:       l.pnt,
		fields:    l.fields,
		calldepth: l.calldepth,
	}
}

func (l *Logger) log(lvl log.Level, tag, msg string, fields ...log.Field) {
	if l.level > lvl {
		return
	}

	// Get file and line number
	_, file, line, ok := runtime.Caller(l.calldepth + 1)
	if ok {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
	} else {
		file = "???"
		line = 0
	}

	ctx := log.Ctx{
		Level:     lvl.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Service:   l.service,
		File:      fmt.Sprintf("%s:%d", file, line),
	}

	f, err := l.fmt.Format(&ctx, tag, msg, fields...)
	if err != nil {
		f = fmt.Sprintf("log formatter error <%s>", err)
	}

	l.pnt.Print(&ctx, f)
}

// Config contains all log-related configuration
type Config struct {
	Level string `toml:"level"`
}
