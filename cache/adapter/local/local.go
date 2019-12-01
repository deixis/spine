// Package local provides an LRU cache and cache-filling library that only runs
// on the local instance.
package local

import (
	"context"
	"sync"

	"github.com/deixis/spine/cache"
	"github.com/deixis/spine/cache/lru"
	"github.com/deixis/spine/config"
)

// Name is the local cache adapter name
const Name = "local"

type localCache struct {
	mu sync.Mutex

	groups map[string]*group
}

// New returns a new local cache
func New(_ config.Tree) (cache.Cache, error) {
	return &localCache{
		groups: make(map[string]*group),
	}, nil
}

func (c *localCache) NewGroup(
	name string, cacheBytes int64, loader cache.LoadFunc,
) cache.Group {
	c.mu.Lock()
	defer c.mu.Unlock()

	g, ok := c.groups[name]
	if !ok {
		g = &group{
			lru:  lru.New(cacheBytes),
			load: loader,
		}
		c.groups[name] = g
	}
	return g
}

type group struct {
	mu sync.Mutex

	lru  *lru.Cache
	load cache.LoadFunc
}

func (g *group) Get(ctx context.Context, key string) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	v, ok := g.lru.Get(key)
	if ok {
		return v.(*vBytes).data, nil
	}

	data, err := g.load(ctx, key)
	if err != nil {
		return nil, err
	}
	g.lru.Set(key, &vBytes{data})
	return data, nil
}

type vBytes struct {
	data []byte
}

func (w *vBytes) Size() int {
	return len(w.data)
}
