// Package main is a service discovery example
//
// It creates an HTTP server and register it to the service discovery agent
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/deixis/spine"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create spine
	config := &AppConfig{}
	app, err := spine.New("disco", config)
	if err != nil {
		fmt.Println("Problem initialising spine", err)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		fmt.Println("Problem parsing port", err)
		os.Exit(1)
	}
	tags := []string{"api", "http", app.Config().Version}

	// Register service
	s := http.NewServer()
	s.HandleFunc("/hello", http.GET, Hello)
	app.RegisterService(&spine.ServiceRegistration{
		Name:   "api.http",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: s,
		Tags:   tags,
	})

	// Listen to service discovery events for that service
	svc, err := app.Disco().Service(app, "api.http", tags...)
	if err != nil {
		fmt.Println("Problem getting service", err)
		os.Exit(1)
	}
	watcher := svc.Watch()
	defer watcher.Close()
	go func() {
		for {
			events, err := watcher.Next()
			if err != nil {
				fmt.Println("Watcher error", err)
				return
			}

			for _, e := range events {
				fmt.Println("Event", e)
			}
		}
	}()

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// Hello handler example
func Hello(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Trace(ctx, "http.hello", "Hello called")
	text := "Hello from spine"
	w.Data(
		http.StatusOK,
		"text/plain",
		ioutil.NopCloser(bytes.NewReader([]byte(text))),
	)
}
