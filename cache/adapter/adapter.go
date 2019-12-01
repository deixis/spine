package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/cache"
	"github.com/deixis/spine/cache/adapter/local"
	"github.com/deixis/spine/config"
)

// Adapter returns a new agent initialised with the given config
type Adapter func(config.Tree) (cache.Cache, error)

var (
	mu       sync.RWMutex
	adapters = make(map[string]Adapter)
)

func init() {
	// Register default adapters
	Register(local.Name, local.New)
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

// Register makes a cache adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()

	if adapter == nil {
		panic("cache: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("cache: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New creates a new cache
func New(config config.Tree) (cache.Cache, error) {
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
	return nil, fmt.Errorf("cache adapter not found <%s>", adapter)
}

// ErrEmptyConfig occurs when initialising a cache from an empty config tree
var ErrEmptyConfig = errors.New("cache config tree is empty")
