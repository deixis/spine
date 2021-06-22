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
	"github.com/deixis/spine/log"
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
		mux:        &sync.Mutex{},
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		reg:        prometheus.NewRegistry(),
		log:        log.NopLogger(),
		Config:     *config,
	}, nil
}

type Client struct {
	mux        *sync.Mutex
	http       http.Server
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	reg        *prometheus.Registry
	log        log.Logger

	Config Config
	Meta   map[string]string
}

func (c *Client) Start() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(c.reg, promhttp.HandlerOpts{}))

	c.http.Addr = strings.Join([]string{c.Config.Addr, c.Config.Port}, ":")
	c.http.Handler = mux

	c.log.Trace("stats.prometheus.http", "Starting Prometheus metrics server",
		log.String("addr", c.http.Addr),
	)

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
	v, err := toFloat64(n)
	if err != nil {
		return
	}

	op, err := c.loadGauge(key, tags...)
	if err != nil {
		c.log.Warning("stats.prometheus.count.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Add(v)
}

func (c *Client) Inc(key string, tags ...map[string]string) {
	// Using gauge instead of Counter because Prometheus uses monotonic counters (no decrement)
	op, err := c.loadGauge(key, tags...)
	if err != nil {
		c.log.Warning("stats.prometheus.inc.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Inc()
}

func (c *Client) Dec(key string, tags ...map[string]string) {
	op, err := c.loadGauge(key, tags...)
	if err != nil {
		c.log.Warning("stats.prometheus.dec.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Dec()
}

func (c *Client) Gauge(key string, n interface{}, tags ...map[string]string) {
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
	v, err := toFloat64(n)
	if err != nil {
		return
	}

	op, err := c.loadGauge(key, tags...)
	if err != nil {
		c.log.Warning("stats.prometheus.gauge.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Set(v)
}

func (c *Client) Timing(key string, t time.Duration, tags ...map[string]string) {
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Histogram
	var tm map[string]string
	if len(tags) > 0 {
		tm = tags[0]
	} else {
		tm = make(map[string]string)
	}

	// Store unit
	tm["unit"] = "ms"
	op, err := c.loadHistogram(key, tm)
	if err != nil {
		c.log.Warning("stats.prometheus.timing.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Observe(float64(t / time.Millisecond))
}

func (c *Client) Histogram(key string, n interface{}, tags ...map[string]string) {
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Histogram
	v, err := toFloat64(n)
	if err != nil {
		return
	}

	op, err := c.loadHistogram(key, tags...)
	if err != nil {
		c.log.Warning("stats.prometheus.histogram.err", "Failed to load collector",
			log.Error(err),
		)
		return
	}
	op.Observe(v)
}

func (c *Client) With(meta map[string]string) stats.Stats {
	clone := c.clone()
	clone.Meta = meta
	return clone
}

func (c *Client) Log(l log.Logger) stats.Stats {
	clone := c.clone()
	clone.log = l
	return clone
}

// clone returns a shallow clone of c
func (c *Client) clone() *Client {
	clone := *c
	return &clone
}

func (c *Client) loadGauge(key string, tags ...map[string]string) (prometheus.Gauge, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	// TODO: Use hash
	// TODO: Use LRU cache (register/unregister metrics when evicted)
	id := buildID(key, tags...)
	if v, ok := c.gauges[id]; ok {
		if len(tags) > 0 {
			return v.With(tags[0]), nil
		}
		return v.With(nil), nil
	}

	// Build collector with or without labels
	var coll *prometheus.GaugeVec
	if len(tags) > 0 {
		coll = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: c.Config.Namespace,
				Name:      sanitiseName(key),
			},
			labels(tags[0]),
		)
	} else {
		coll = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: c.Config.Namespace,
				Name:      sanitiseName(key),
			},
			nil,
		)
	}

	// Register it
	if err := c.reg.Register(uncheckedCollector{coll}); err != nil {
		return nil, err
	}

	// Cache it
	c.gauges[id] = coll

	if len(tags) > 0 {
		return coll.With(tags[0]), nil
	}
	return coll.With(nil), nil
}

func (c *Client) loadHistogram(key string, tags ...map[string]string) (prometheus.Observer, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	// TODO: Use hash
	// TODO: Use LRU cache (register/unregister metrics when evicted)
	id := buildID(key, tags...)
	if v, ok := c.histograms[id]; ok {
		if len(tags) > 0 {
			return v.With(tags[0]), nil
		}
		return v.With(nil), nil
	}

	// Build collector with or without labels
	var coll *prometheus.HistogramVec
	if len(tags) > 0 {
		coll = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: c.Config.Namespace,
				Name:      sanitiseName(key),
			},
			labels(tags[0]),
		)
	} else {
		coll = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: c.Config.Namespace,
				Name:      sanitiseName(key),
			},
			nil,
		)
	}

	// Register it
	if err := c.reg.Register(uncheckedCollector{coll}); err != nil {
		return nil, err
	}

	// Cache it
	c.histograms[id] = coll

	if len(tags) > 0 {
		return coll.With(tags[0]), nil
	}
	return coll.With(nil), nil
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

// uncheckedCollector wraps a Collector but its Describe method yields no Desc.
// This allows incoming metrics to have inconsistent label sets
type uncheckedCollector struct {
	c prometheus.Collector
}

func (u uncheckedCollector) Describe(_ chan<- *prometheus.Desc) {}
func (u uncheckedCollector) Collect(c chan<- prometheus.Metric) {
	u.c.Collect(c)
}
