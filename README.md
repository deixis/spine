# Spine

Spine is a backend building block written in Go.

## Packages

1. [Background](./bg)
1. [Cache](./cache)
1. [Config](./config)
1. [Context](./context)
1. [Crypto](./crypto)
1. [Disco](./disco)
1. [Log](./log)
1. [Net](./net)
1. [Schedule](./schedule)
1. [Stats](./stats)
1. [Testing](./testing)
1. [Tracing](./tracing)

## Demo

Start a simple HTTP server

```shell
$ git clone https://github.com/deixis/spine.git
$ cd spine/example
$ CONFIG_URI=file://${PWD}/config.toml go run http_server.go
```

Send a request

```shell
$ curl -v http://127.0.0.1:3000/ping
```

## Example

### Simple HTTP server

This code creates a SPINE instance and attach and HTTP handler to it with one route `/ping`.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/deixis/spine"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
)

var (
	version = "dirty"
	date    = "now"
)

type Config struct {
	Foo string `toml:"foo"`
}

func main() {
	// Create spine
	appConfig := &Config{}
	app, err := spine.New("demo", appConfig)
	if err != nil {
		return errors.Wrap(err, "error initialising spine")
	}
	app.Config().Version = version

	// Register HTTP handler
	httpServer := http.NewServer()
	httpServer.HandleFunc("/ping", http.GET, Ping)
	app.RegisterService(&spine.ServiceRegistration{
		Name:   "http.demo",
		Host:   os.Getenv("IP"),
		Port:   8080,
		Server: httpServer,
		Tags:   []string{"http"},
	})

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

```

### Config

Example of a configuration file

```toml
node = "$HOSTNAME"
version = "1.1"

[request]
  timeout_ms = 1000
  allow_context = false

[disco.consul]
  address = "localhost:7500"
  dc = "local"

[log.printer.stdout]

[cache.local]

[app.demo]
  foo = "bar"
```
