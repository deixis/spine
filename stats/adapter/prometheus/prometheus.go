package prometheus

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/deixis/spine/config"
	"github.com/deixis/spine/stats"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const Name = "prometheus"

type Config struct {
	Addr string            `toml:"addr"`
	Port string            `toml:"port"`
	Tags map[string]string `toml:"tags"`
}

func New(tree config.Tree) (stats.Stats, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, err
	}

	return &Client{
		reg:    prometheus.NewRegistry(),
		Config: *config,
	}, nil
}

type Client struct {
	http     http.Server
	counters sync.Map
	reg      *prometheus.Registry

	Config Config
	Meta   map[string]string
}

func (c *Client) Start() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	c.http.Addr = strings.Join([]string{c.Config.Addr, c.Config.Port}, ":")
	c.http.Handler = mux

	err := c.http.ListenAndServe()
	if err == http.ErrServerClosed {
		// Suppress error caused by a server Shutdown or Close
		return
	}
	// FIXME: Refactor stats to return error on Start()
	panic(errors.Wrap(err, "failed to start stats/prometheus http server"))
}
func (c *Client) Stop() {
	c.http.Shutdown(context.Background()) // Close all idle connections
}

func (c *Client) Count(key string, n interface{}, tags ...map[string]string) {
	// TODO: Implement me
	// op := c.loadCounter(key)
}

func (c *Client) Inc(key string, tags ...map[string]string) {
	// TODO: Inlude tags
	// TODO: Cache counter (POOL LRU CACHE)
	op := c.loadCounter(key)
	op.Inc()
}

func (c *Client) Dec(key string, tags ...map[string]string) {
	// TODO: Implement me
}

func (c *Client) Gauge(key string, n interface{}, tags ...map[string]string) {
	// TODO: Implement me
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
}

func (c *Client) Timing(key string, t time.Duration, tags ...map[string]string) {
	// TODO: Implement me
}

func (c *Client) Histogram(key string, n interface{}, tags ...map[string]string) {
	// TODO: Implement me
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Histogram
}

func (c *Client) With(meta map[string]string) stats.Stats {
	return &Client{
		http:     c.http,
		reg:      c.reg,
		counters: c.counters,
		Config:   c.Config,
		Meta:     meta,
	}
}

func (c *Client) loadCounter(key string) prometheus.Counter {
	v, ok := c.counters.Load(key)
	if !ok {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: strings.ReplaceAll(key, ".", "_"),
		})
		if err := c.reg.Register(counter); err != nil {
			panic(err)
		}
		c.counters.Store(key, counter)
		return counter
	}
	return v.(prometheus.Counter)
}
