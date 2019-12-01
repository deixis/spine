package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/stats"
	"github.com/deixis/spine/stats/adapter/statsd"
)

func init() {
	Register(statsd.Name, statsd.New)
}

// Adapter returns a new store initialised with the given config
type Adapter func(config config.Tree) (stats.Stats, error)

var (
	adaptersMu sync.RWMutex
	adapters   = make(map[string]Adapter)
)

// Adapters returns the list of registered adapters
func Adapters() []string {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	var l []string
	for a := range adapters {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a stats adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()

	if adapter == nil {
		panic("stats: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("stats: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New returns a new stats instance
func New(config config.Tree) (stats.Stats, error) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	keys := config.Keys()
	if len(keys) == 0 {
		return nil, ErrEmptyConfig
	}
	adapter := keys[0]

	if f, ok := adapters[adapter]; ok {
		return f(config.Get(adapter))
	}
	return nil, fmt.Errorf("stats adapter not found <%s>", adapter)
}

// ErrEmptyConfig occurs when initialising stats from an empty config tree
var ErrEmptyConfig = errors.New("stats config tree is empty")
