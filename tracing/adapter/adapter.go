package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/tracing"
	"github.com/deixis/spine/tracing/adapter/jaeger"
)

// Adapter returns a new agent initialised with the given config
type Adapter func(config.Tree, ...tracing.TracerOption) (tracing.Tracer, error)

var (
	mu       sync.RWMutex
	adapters = make(map[string]Adapter)
)

func init() {
	// Register default adapters
	Register(jaeger.Name, jaeger.New)
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

// Register makes a tracing adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()

	if adapter == nil {
		panic("tracing: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("tracing: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New creates a new tracing
func New(config config.Tree, o ...tracing.TracerOption) (tracing.Tracer, error) {
	mu.RLock()
	defer mu.RUnlock()

	keys := config.Keys()
	if len(keys) == 0 {
		return nil, ErrEmptyConfig
	}
	adapter := keys[0]

	if f, ok := adapters[adapter]; ok {
		return f(config.Get(adapter), o...)
	}
	return nil, fmt.Errorf("tracing adapter not found <%s>", adapter)
}

// ErrEmptyConfig occurs when initialising tracer from an empty config tree
var ErrEmptyConfig = errors.New("tracer config tree is empty")
