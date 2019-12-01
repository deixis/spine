// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/deixis/spine"
	"github.com/deixis/spine/cache"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func main() {
	// Create spine
	config := &AppConfig{}
	app, err := spine.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising spine", err)
		os.Exit(1)
	}

	// Cache random data
	grp := app.Cache().NewGroup("foo", 64<<20, cache.LoadFunc(
		func(ctx context.Context, key string) ([]byte, error) {
			log.FromContext(ctx).Warning("cache.load", "Filling cache...", log.String("key", key))
			return []byte(app.Config().Node), nil
		},
	))
	go func() {
		for {
			if app.Err() != nil {
				return
			}
			key := genRandomString()
			ctx := context.Background()
			v, err := grp.Get(ctx, key)
			if err != nil {
				app.L().Error("example.cache.err", "Error loading data",
					log.Error(err),
				)
				continue
			}

			fmt.Println(app.Config().Node, "got key", key, "from node", string(v))
			time.Sleep(time.Second * time.Duration(rand.Int63n(15)))
		}
	}()

	// Register HTTP handler
	h := handler{
		cache: grp,
	}
	s := http.NewServer()
	s.HandleFunc("/cache/{key}", http.GET, h.Load)
	app.RegisterServer("127.0.0.1:3000", s)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

func genRandomString() string {
	b := make([]byte, 1)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type handler struct {
	cache cache.Group
}

// Cache handler example
func (h *handler) Load(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Trace(ctx, "http.cache.load", "Load data", log.String("key", r.Params["key"]))
	v, err := h.cache.Get(ctx, r.Params["key"])
	if err != nil {
		log.Warn(ctx, "http.cache.err", "Error pulling data from cache", log.Error(err))
		w.Head(http.StatusInternalServerError)
		return
	}
	w.Data(http.StatusOK, "text/plain", ioutil.NopCloser(bytes.NewReader(v)))
}
