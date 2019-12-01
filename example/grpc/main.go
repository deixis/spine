// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/deixis/spine"
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
	app, err := spine.New("grpc", config)
	if err != nil {
		fmt.Println("Problem initialising spine", err)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		fmt.Println("Problem parsing port", err)
		os.Exit(1)
	}

	hs := http.NewServer()

	api := &publicAPI{}
	hs.HandleFunc("hello", http.GET, api.Hello)

	app.RegisterService(&spine.ServiceRegistration{
		Name:   "api.cache",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: hs,
	})

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

type publicAPI struct{}

func (h *publicAPI) Hello(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Trace(ctx, "http.hello", "Hello")
}
