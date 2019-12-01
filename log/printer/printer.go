package printer

import (
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/log/printer/file"
	"github.com/deixis/spine/log/printer/stdout"
)

func init() {
	Register(stdout.Name, stdout.New)
	Register(file.Name, file.New)
}

// Printer returns a new logger initialised with the given config
type Printer func(config config.Tree) (log.Printer, error)

var (
	printersMu sync.RWMutex
	printers   = make(map[string]Printer)
)

// Printers returns the list of registered printers
func Printers() []string {
	printersMu.RLock()
	defer printersMu.RUnlock()

	var l []string
	for a := range printers {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a logger printer available by the provided name.
// If an printer is registered twice or if an printer is nil, it will panic.
func Register(name string, printer Printer) {
	printersMu.Lock()
	defer printersMu.Unlock()

	if printer == nil {
		panic("logs: Registered printer is nil")
	}
	if _, dup := printers[name]; dup {
		panic("logs: Duplicated printer")
	}

	printers[name] = printer
}

// New returns a new logger instance
func New(config config.Tree) (log.Printer, error) {
	printersMu.RLock()
	defer printersMu.RUnlock()

	var adapter string
	if len(config.Keys()) > 0 {
		adapter = config.Keys()[0]
	}

	if adapter == "" {
		return stdout.New(config.Get(stdout.Name))
	}

	if f, ok := printers[adapter]; ok {
		return f(config.Get(adapter))
	}
	return nil, fmt.Errorf("log printer not found <%s>", adapter)
}
