package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/deixis/spine"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/example/grpc/server/demo"
	"github.com/deixis/spine/log"
	lgrpc "github.com/deixis/spine/net/grpc"
	"github.com/deixis/spine/net/naming"
	"google.golang.org/grpc"
)

func main() {
	err := start()
	if err != nil {
		fmt.Println("App error:", err)
		os.Exit(1)
	}
}

type AppConfig struct {
	Foo string `json:"foo"`
}

func start() error {
	// Create spine
	config := &AppConfig{}
	app, err := spine.New("grpc-client", config)
	if err != nil {
		return errors.Wrap(err, "error initialising spine")
	}

	// Setup gRPC client
	c, err := lgrpc.NewClient(
		app,
		"disco://grpc.demo",
		grpc.WithInsecure(),
		grpc.WithTimeout(time.Second*10),
		grpc.WithBlock(),
		grpc.WithBalancer(grpc.RoundRobin(
			lgrpc.WrapResolver(naming.URI(app)),
		)),
	)
	if err != nil {
		return errors.Wrap(err, "error connecting to server")
	}
	c.PropagateContext = true

	// Setup demo service
	demoSvc := demo.NewDemoClient(c.GRPC)

	// Call service
	for i := 0; i < 3; i++ {
		// Prepare context
		ctx := context.Background()
		log.Trace(ctx, "prepare", "Prepare context")
		ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
		ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
		ctx = lcontext.WithShipment(ctx, "flag", 3)

		res, err := demoSvc.Hello(ctx, &demo.Request{Msg: "Ping"})
		if err != nil {
			return errors.Wrap(err, "grpc call failed")
		}
		fmt.Println("Hello service returned", res.Msg)
	}
	return nil
}
