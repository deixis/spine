package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/net/pubsub"
	"github.com/deixis/spine/net/pubsub/adapter/inmem"
)

// Adapter returns a new agent initialised with the given config
type Adapter func(config.Tree) (pubsub.PubSub, error)

var (
	mu       sync.RWMutex
	adapters = make(map[string]Adapter)
)

func init() {
	// Register default adapters
	Register(inmem.Name, inmem.New)
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

// Register makes a pubsub adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()

	if adapter == nil {
		panic("pubsub: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("pubsub: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New creates a new pubsub
func New(config config.Tree) (pubsub.PubSub, error) {
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
	return nil, fmt.Errorf("pubsub adapter not found <%s>", adapter)
}

// ErrEmptyConfig occurs when initialising pubsub from an empty config tree
var ErrEmptyConfig = errors.New("pubsub config tree is empty")
