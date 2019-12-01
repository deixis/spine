package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/deixis/spine"
	"github.com/deixis/spine/example/grpc/server/demo"
	"github.com/deixis/spine/log"
	lgrpc "github.com/deixis/spine/net/grpc"
	"google.golang.org/grpc"
)

func main() {
	err := start()
	if err != nil {
		fmt.Println("App error", err)
		os.Exit(1)
	}
}

type AppConfig struct {
	Foo string `json:"foo"`
}

func start() error {
	// Create spine
	config := &AppConfig{}
	app, err := spine.New("grpc-server", config)
	if err != nil {
		return errors.Wrap(err, "Problem initialising spine")
	}

	// Build gRPC server
	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		return errors.Wrap(err, "Problem parsing port")
	}
	s := lgrpc.NewServer()
	s.AppendUnaryMiddleware(traceMiddleware)
	s.Handle(func(s *grpc.Server) {
		demo.RegisterDemoServer(s, &gRPCServer{
			node: os.Getenv("NODE_NAME"),
		})
	})

	// Register gRPC handler as a service
	app.RegisterService(&spine.ServiceRegistration{
		Name:   "grpc.demo",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: s,
	})

	// Start serving requests
	err = app.Serve()
	if err != nil {
		return errors.Wrap(err, "Problem serving requests")
	}
	return nil
}

type gRPCServer struct {
	node string
}

func (s *gRPCServer) Hello(
	ctx context.Context, req *demo.Request,
) (*demo.Response, error) {
	log.Trace(ctx, "grpc.hello", "Calling Hello", log.String("node", s.node))

	return &demo.Response{Msg: s.node}, nil
}

func traceMiddleware(next lgrpc.UnaryHandler) lgrpc.UnaryHandler {
	return func(ctx context.Context, info *lgrpc.Info, req interface{}) (interface{}, error) {
		log.Trace(ctx, "grpc.trace.start", "Start call")
		res, err := next(ctx, info, req)
		log.Trace(ctx, "grpc.trace.end", "End call")
		return res, err
	}
}
