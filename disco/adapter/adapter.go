package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/disco"
	"github.com/deixis/spine/disco/adapter/consul"
)

// Adapter returns a new agent initialised with the given config
type Adapter func(config.Tree) (disco.Agent, error)

var (
	mu       sync.RWMutex
	adapters = make(map[string]Adapter)
)

func init() {
	// Register default adapters
	Register(consul.Name, consul.New)
}

// Adapters returns the list of registered adapters
func Adapters() []string {
	mu.RLock()
	defer mu.RUnlock()

	var l []string
	for a := range adapters {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a registrar adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()

	if adapter == nil {
		panic("disco: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("disco: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New creates a new service discovery agent
func New(config config.Tree) (disco.Agent, error) {
	mu.RLock()
	defer mu.RUnlock()

	keys := config.Keys()
	if len(keys) == 0 {
		return nil, ErrEmptyConfig
	}
	adapter := keys[0]

	if f, ok := adapters[adapter]; ok {
		return f(config.Get(adapter))
	}
	return nil, fmt.Errorf("disco adapter not found <%s>", adapter)
}

// ErrEmptyConfig occurs when initialising a disco from an empty config tree
var ErrEmptyConfig = errors.New("disco config tree is empty")
