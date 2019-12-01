package main

import (
	"context"
	"fmt"
	"os"

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
	app, err := spine.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising spine", err)
		os.Exit(1)
	}

	// Register HTTP handler
	s := http.NewServer()
	s.HandleFunc("/ping", http.GET, Ping)
	app.RegisterServer("127.0.0.1:3000", s)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// Ping handler example
func Ping(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Trace(ctx, "action.ping", "Simple request", log.String("ua", r.HTTP.UserAgent()))
	w.Head(http.StatusOK)
}
