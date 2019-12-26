// Package stdout prints log lines into the standard output.
// It also colorised outputs with ANSI Escape Codes
package stdout

import (
	"fmt"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/fatih/color"
)

const Name = "stdout"

var (
	traceColour   = color.New(color.FgBlue)
	warningColour = color.New(color.FgYellow)
	errorColour   = color.New(color.FgRed)
	unknownColour = color.New(color.FgWhite)
)

func New(c config.Tree) (log.Printer, error) {
	return &Logger{}, nil
}

type Logger struct{}

func (l *Logger) Print(ctx *log.Context, s string) error {
	colour := pickColour(ctx.Level)
	fmt.Println(colour.SprintFunc()(s))
	return nil
}

func (l *Logger) Close() error {
	return nil
}

func pickColour(lvl log.Level) *color.Color {
	switch lvl {
	case log.LevelTrace:
		return traceColour
	case log.LevelWarning:
		return warningColour
	case log.LevelError:
		return errorColour
	}

	return unknownColour
}
