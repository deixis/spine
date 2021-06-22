// Package main is a tracing example with Jaeger
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deixis/spine"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
	"github.com/deixis/spine/stats"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	// Create spine
	config := &AppConfig{}
	app, err := spine.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising spine", err)
		os.Exit(1)
	}

	// Register HTTP handler
	h := handler{}
	s := http.NewServer()
	s.HandleFunc("/test", http.GET, h.Test)
	app.RegisterServer("127.0.0.1:3003", s)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

type handler struct {
}

// Cache handler example
func (h *handler) Test(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Stats
	startTime := time.Now()
	stats := stats.FromContext(ctx)
	tags := map[string]string{
		"method": r.HTTP.Method,
		"path":   r.HTTP.URL.Path,
	}
	stats.Inc("http.conc", tags)

	n := seededRand.Intn(2000)
	log.FromContext(ctx).Trace("http.stats.test", "Test request",
		log.Int("wait_ms", n),
	)

	time.Sleep(time.Duration(n) * time.Millisecond)
	w.Data(http.StatusOK, "text/plain", ioutil.NopCloser(strings.NewReader("OK")))

	// Stats
	tags["status"] = strconv.Itoa(w.Code())
	stats.Histogram("http.call", 1, tags)
	stats.Timing("http.time", time.Since(startTime), tags)
	stats.Dec("http.conc", tags)
}
