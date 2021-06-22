package prometheus

import (
	"context"
	"math"
	"net/http"
	"sort"
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
	Namespace string            `toml:"namespace"`
	Addr      string            `toml:"addr"`
	Port      string            `toml:"port"`
	Tags      map[string]string `toml:"tags"`
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
	http       http.Server
	counters   sync.Map
	gauges     sync.Map
	histograms sync.Map
	reg        *prometheus.Registry

	Config Config
	Meta   map[string]string
}

func (c *Client) Start() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(c.reg, promhttp.HandlerOpts{}))

	c.http.Addr = strings.Join([]string{c.Config.Addr, c.Config.Port}, ":")
	c.http.Handler = mux

	err := c.http.ListenAndServe()
	if err == http.ErrServerClosed {
		// Suppress error caused by a server Shutdown or Close
		return
	}
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
	op := c.loadCounter(key, tags...)
	op.Inc()
}

func (c *Client) Dec(key string, tags ...map[string]string) {
	// TODO: Implement me
}

func (c *Client) Gauge(key string, n interface{}, tags ...map[string]string) {
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
	v, err := toFloat64(n)
	if err != nil {
		return
	}

	op := c.loadGauge(key, tags...)
	op.Set(v)
}

func (c *Client) Timing(key string, t time.Duration, tags ...map[string]string) {
	// TODO: Implement me
}

func (c *Client) Histogram(key string, n interface{}, tags ...map[string]string) {
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Histogram
	v, err := toFloat64(n)
	if err != nil {
		return
	}

	op := c.loadHistogram(key, tags...)
	op.Observe(v)
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

func (c *Client) loadCounter(key string, tags ...map[string]string) prometheus.Counter {
	id := buildID(key, tags...)

	v, ok := c.counters.Load(id)
	if !ok {
		// Build collector with or without labels
		var coll prometheus.Collector
		if len(tags) > 0 {
			coll = prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
				labels(tags[0]),
			)
		} else {
			coll = prometheus.NewCounter(
				prometheus.CounterOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
			)
		}

		// Register it
		if err := c.reg.Register(coll); err != nil {
			panic(err)
		}

		// Cache it
		c.counters.Store(id, coll)

		v = coll
	}

	if len(tags) > 0 {
		return v.(*prometheus.CounterVec).With(tags[0])
	}
	return v.(prometheus.Counter)
}

func (c *Client) loadGauge(key string, tags ...map[string]string) prometheus.Gauge {
	id := buildID(key, tags...)

	v, ok := c.gauges.Load(id)
	if !ok {
		// Build collector with or without labels
		var coll prometheus.Collector
		if len(tags) > 0 {
			coll = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
				labels(tags[0]),
			)
		} else {
			coll = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
			)
		}

		// Register it
		if err := c.reg.Register(coll); err != nil {
			panic(err)
		}

		// Cache it
		c.gauges.Store(id, coll)

		v = coll
	}

	if len(tags) > 0 {
		return v.(*prometheus.GaugeVec).With(tags[0])
	}
	return v.(prometheus.Gauge)
}

func (c *Client) loadHistogram(key string, tags ...map[string]string) prometheus.Observer {
	id := buildID(key, tags...)

	v, ok := c.histograms.Load(id)
	if !ok {
		// Build collector with or without labels
		var coll prometheus.Collector
		if len(tags) > 0 {
			coll = prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
				labels(tags[0]),
			)
		} else {
			coll = prometheus.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: c.Config.Namespace,
					Name:      sanitiseName(key),
				},
			)
		}

		// Register it
		if err := c.reg.Register(coll); err != nil {
			panic(err)
		}

		// Cache it
		c.histograms.Store(id, coll)

		v = coll
	}

	if len(tags) > 0 {
		return v.(*prometheus.HistogramVec).With(tags[0])
	}
	return v.(prometheus.Histogram)
}

func buildID(key string, tags ...map[string]string) string {
	key = sanitiseName(key)
	if len(tags) == 0 {
		return key
	}

	labels := labels(tags[0])
	sort.Strings(labels) // Ensure consistent naming
	return strings.Join(append([]string{key}, labels...), ".")
}

func labels(tags map[string]string) []string {
	labels := make([]string, 0, len(tags))
	for k := range tags {
		labels = append(labels, k)
	}
	return labels
}

// sanitiseName removes unauthorised characters for Prometheus metrics keys
func sanitiseName(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}

func toFloat64(n interface{}) (float64, error) {
	var v float64
	switch n := n.(type) {
	case float64:
		v = n
	case float32:
		v = float64(n)
	case int:
		v = float64(n)
	case int8:
		v = float64(n)
	case int16:
		v = float64(n)
	case int32:
		v = float64(n)
	case int64:
		v = float64(n)
	default:
		// NaN
		return math.NaN(), errors.New("failed to convert value to float64")
	}
	return v, nil
}
