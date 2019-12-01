// Package main is a tracing example with Jaeger
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/deixis/spine"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
	"github.com/deixis/spine/tracing"
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
	s.HandleFunc("/trace", http.GET, h.Trace)
	app.RegisterServer("127.0.0.1:3000", s)

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
func (h *handler) Trace(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	n := seededRand.Intn(2000)
	log.FromContext(ctx).Trace("http.tracing.trace", "Trace request",
		log.Int("wait_ms", n),
	)

	span := tracing.FromContext(ctx).StartSpan("http_request")
	defer span.Finish()
	// span, _ := opentracing.StartSpanFromContext(ctx, "http_request")
	// defer span.Finish()

	time.Sleep(time.Duration(n) * time.Millisecond)
	w.Data(http.StatusOK, "text/plain", ioutil.NopCloser(strings.NewReader("OK")))
}
