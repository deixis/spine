package statsd

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

// Source : https://github.com/alexcesaro/statsd/blob/master/conn.go

type conn struct {
	// Fields settable with options at Client's creation.
	addr          string
	errorHandler  func(error)
	flushPeriod   time.Duration
	maxPacketSize int
	network       string
	tagFormat     TagFormat

	mu sync.Mutex
	// Fields guarded by the mutex.
	closed    bool
	w         io.WriteCloser
	buf       []byte
	rateCache map[float32]string
}

func newConn(conf connConfig) (*conn, error) {
	c := &conn{
		addr:          conf.Addr,
		errorHandler:  conf.ErrorHandler,
		flushPeriod:   conf.FlushPeriod,
		maxPacketSize: conf.MaxPacketSize,
		network:       conf.Network,
		tagFormat:     conf.TagFormat,
		// Discard writes until the connection is successfuly established
		w: &nopWriter{io.Discard},
	}

	// Connection check to fail as early as possible
	var checkConn = noopCheck
	if c.network[:3] == "udp" {
		checkConn = checkUDP
	}

	// First attempt to connect synchronously to ensure we have a chance to
	// forward the first metrics received
	// Connect to statsd server
	writer, err := dialTimeout(c.network, c.addr, 5*time.Second)
	if err == nil {
		err = checkConn(writer)
	}
	if err != nil {
		// TODO: Use structured log format
		fmt.Printf("[statsd] failed to dial connection to <%s>: %s\n", c.addr, err)

		if writer != nil {
			writer.Close()
			writer = nil
		}

		// Periodically attempt to connect in background
		go func() {
			for {
				// TODO: Use exponential backoff instead
				time.Sleep(10 * time.Second)

				fmt.Println("[statsd] Dialing...")

				// Connect to statsd server
				writer, err := dialTimeout(c.network, c.addr, 5*time.Second)
				if err != nil {
					fmt.Printf("[statsd] failed to dial connection to <%s>: %s\n", c.addr, err)

					continue
				}

				if err := checkConn(writer); err != nil {
					fmt.Printf("[statsd] connection check failed: %s\n", err)
					writer.Close()

					continue
				}

				// Replace writer
				c.w = writer

				return
			}
		}()
	}

	// Replace writer
	if writer != nil {
		c.w = writer
	}

	// To prevent a buffer overflow add some capacity to the buffer to allow for
	// an additional metric.
	c.buf = make([]byte, 0, c.maxPacketSize+200)

	if c.flushPeriod > 0 {
		go func() {
			ticker := time.NewTicker(c.flushPeriod)
			for range ticker.C {
				c.mu.Lock()
				if c.closed {
					ticker.Stop()
					c.mu.Unlock()
					return
				}
				c.flush(0)
				c.mu.Unlock()
			}
		}()
	}

	return c, nil
}

func (c *conn) metric(prefix, bucket string, n interface{}, typ string, rate float32, tags string) {
	c.mu.Lock()
	l := len(c.buf)
	c.appendBucket(prefix, bucket, tags)
	c.appendNumber(n)
	c.appendType(typ)
	c.appendRate(rate)
	c.closeMetric(tags)
	c.flushIfBufferFull(l)
	c.mu.Unlock()
}

func (c *conn) gauge(prefix, bucket string, value interface{}, tags string) {
	c.mu.Lock()
	l := len(c.buf)
	// To set a gauge to a negative value we must first set it to 0.
	// https://github.com/etsy/statsd/blob/master/docs/metric_types.md#gauges
	if isNegative(value) {
		c.appendBucket(prefix, bucket, tags)
		c.appendGauge(0, tags)
	}
	c.appendBucket(prefix, bucket, tags)
	c.appendGauge(value, tags)
	c.flushIfBufferFull(l)
	c.mu.Unlock()
}

func (c *conn) appendGauge(value interface{}, tags string) {
	c.appendNumber(value)
	c.appendType("g")
	c.closeMetric(tags)
}

func (c *conn) unique(prefix, bucket string, value string, tags string) {
	c.mu.Lock()
	l := len(c.buf)
	c.appendBucket(prefix, bucket, tags)
	c.appendString(value)
	c.appendType("s")
	c.closeMetric(tags)
	c.flushIfBufferFull(l)
	c.mu.Unlock()
}

func (c *conn) appendByte(b byte) {
	c.buf = append(c.buf, b)
}

func (c *conn) appendString(s string) {
	c.buf = append(c.buf, s...)
}

func (c *conn) appendNumber(v interface{}) {
	switch n := v.(type) {
	case int:
		c.buf = strconv.AppendInt(c.buf, int64(n), 10)
	case uint:
		c.buf = strconv.AppendUint(c.buf, uint64(n), 10)
	case int64:
		c.buf = strconv.AppendInt(c.buf, n, 10)
	case uint64:
		c.buf = strconv.AppendUint(c.buf, n, 10)
	case int32:
		c.buf = strconv.AppendInt(c.buf, int64(n), 10)
	case uint32:
		c.buf = strconv.AppendUint(c.buf, uint64(n), 10)
	case int16:
		c.buf = strconv.AppendInt(c.buf, int64(n), 10)
	case uint16:
		c.buf = strconv.AppendUint(c.buf, uint64(n), 10)
	case int8:
		c.buf = strconv.AppendInt(c.buf, int64(n), 10)
	case uint8:
		c.buf = strconv.AppendUint(c.buf, uint64(n), 10)
	case float64:
		c.buf = strconv.AppendFloat(c.buf, n, 'f', -1, 64)
	case float32:
		c.buf = strconv.AppendFloat(c.buf, float64(n), 'f', -1, 32)
	}
}

func isNegative(v interface{}) bool {
	switch n := v.(type) {
	case int:
		return n < 0
	case uint:
		return n < 0
	case int64:
		return n < 0
	case uint64:
		return n < 0
	case int32:
		return n < 0
	case uint32:
		return n < 0
	case int16:
		return n < 0
	case uint16:
		return n < 0
	case int8:
		return n < 0
	case uint8:
		return n < 0
	case float64:
		return n < 0
	case float32:
		return n < 0
	}
	return false
}

func (c *conn) appendBucket(prefix, bucket string, tags string) {
	c.appendString(prefix)
	c.appendString(".") // Separator
	c.appendString(bucket)
	if c.tagFormat == InfluxDB {
		c.appendString(tags)
	}
	c.appendByte(':')
}

func (c *conn) appendType(t string) {
	c.appendByte('|')
	c.appendString(t)
}

func (c *conn) appendRate(rate float32) {
	if rate == 1 {
		return
	}
	if c.rateCache == nil {
		c.rateCache = make(map[float32]string)
	}

	c.appendString("|@")
	if s, ok := c.rateCache[rate]; ok {
		c.appendString(s)
	} else {
		s = strconv.FormatFloat(float64(rate), 'f', -1, 32)
		c.rateCache[rate] = s
		c.appendString(s)
	}
}

func (c *conn) closeMetric(tags string) {
	if c.tagFormat == Datadog {
		c.appendString(tags)
	}
	c.appendByte('\n')
}

func (c *conn) flushIfBufferFull(lastSafeLen int) {
	if len(c.buf) > c.maxPacketSize {
		c.flush(lastSafeLen)
	}
}

// flush flushes the first n bytes of the buffer.
// If n is 0, the whole buffer is flushed.
func (c *conn) flush(n int) {
	if len(c.buf) == 0 {
		return
	}
	if n == 0 {
		n = len(c.buf)
	}

	// Trim the last \n, StatsD does not like it.
	_, err := c.w.Write(c.buf[:n-1])
	c.handleError(err)
	if n < len(c.buf) {
		copy(c.buf, c.buf[n:])
	}
	c.buf = c.buf[:len(c.buf)-n]
}

func (c *conn) handleError(err error) {
	if err != nil && c.errorHandler != nil {
		c.errorHandler(err)
	}
}

var noopCheck = func(w io.Writer) error { return nil }

func checkUDP(w io.Writer) error {
	// When using UDP do a quick check to see if something is listening on the
	// given port to return an error as soon as possible.
	for i := 0; i < 2; i++ {
		_, err := w.Write(nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stubbed out for testing.
var (
	dialTimeout = net.DialTimeout
	now         = time.Now
	randFloat   = rand.Float32
)

type nopWriter struct {
	w io.Writer
}

func (n *nopWriter) Write(b []byte) (int, error) {
	return n.w.Write(b)
}

func (n *nopWriter) Close() error {
	return nil
}
