package statsd

import (
	"bytes"
	"time"
)

// TagFormat represents the format of tags sent by a Client.
type TagFormat uint8

const (
	// InfluxDB tag format.
	// See https://influxdb.com/blog/2015/11/03/getting_started_with_influx_statsd.html
	InfluxDB TagFormat = iota + 1
	// Datadog tag format.
	// See http://docs.datadoghq.com/guides/metrics/#tags
	Datadog
)

type tag struct {
	K, V string
}

type Config struct {
	Addr       string            `toml:"addr"`
	Port       string            `toml:"port"`
	TagsFormat string            `toml:"tags_format"`
	Tags       map[string]string `toml:"tags"`
}

type adapterConfig struct {
	Client clientConfig
	Conn   connConfig
}

type clientConfig struct {
	Rate   float32
	Prefix string
	Tags   []tag
}

type connConfig struct {
	Addr          string
	ErrorHandler  func(error)
	FlushPeriod   time.Duration
	MaxPacketSize int
	Network       string
	TagFormat     TagFormat
}

var joinFuncs = map[TagFormat]func([]tag) string{
	// InfluxDB tag format: ,tag1=payroll,region=us-west
	// https://influxdb.com/blog/2015/11/03/getting_started_with_influx_statsd.html
	InfluxDB: func(tags []tag) string {
		var buf bytes.Buffer
		for _, tag := range tags {
			_ = buf.WriteByte(',')
			_, _ = buf.WriteString(tag.K)
			_ = buf.WriteByte('=')
			_, _ = buf.WriteString(tag.V)
		}
		return buf.String()
	},
	// Datadog tag format: |#tag1:value1,tag2:value2
	// http://docs.datadoghq.com/guides/dogstatsd/#datagram-format
	Datadog: func(tags []tag) string {
		buf := bytes.NewBufferString("|#")
		first := true
		for _, tag := range tags {
			if first {
				first = false
			} else {
				_ = buf.WriteByte(',')
			}
			_, _ = buf.WriteString(tag.K)
			_ = buf.WriteByte(':')
			_, _ = buf.WriteString(tag.V)
		}
		return buf.String()
	},
}
