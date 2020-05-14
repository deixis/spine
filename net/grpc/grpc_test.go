package grpc_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/deixis/spine/tracing"

	"github.com/deixis/spine/config"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	lgrpc "github.com/deixis/spine/net/grpc"
	lt "github.com/deixis/spine/testing"
	"google.golang.org/grpc"
)

func TestClientServer(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c.PropagateContext = true
	if err := c.WaitForStateReady(ctx); err != nil {
		t.Fatal(err)
	}
	testClient := NewTestClient(c.GRPC)

	ctx, _ = lcontext.NewTransitWithContext(ctx)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	res, err := testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	expectMsg := "Pong"
	if expectMsg != res.Msg {
		t.Errorf("expect msg to be %s, but got %s", expectMsg, res.Msg)
	}

	h.Drain()
}

func TestStream(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c.AppendStreamMiddleware(lgrpc.OpenTracingStreamClientMiddleware())
	c.PropagateContext = true
	if err := c.WaitForStateReady(ctx); err != nil {
		t.Fatal(err)
	}
	testClient := NewTestClient(c.GRPC)

	ctx, _ = lcontext.NewTransitWithContext(ctx)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	flowClient, err := testClient.HelloFlow(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := flowClient.Send(&Request{Msg: "Ping"}); err != nil {
		t.Fatal(err)
	}
	if err := flowClient.CloseSend(); err != nil {
		t.Fatal(err)
	}

	for {
		res, err := flowClient.Recv()
		switch err {
		case nil, io.EOF:
		default:
			t.Fatal(err)
		}
		if err == io.EOF {
			break
		}

		expectMsg := "Pong"
		if expectMsg != res.Msg {
			t.Errorf("expect msg to be %s, but got %s", expectMsg, res.Msg)
		}
	}

	h.Drain()
}

func TestClientServerWithTLS(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	h.ActivateTLS("./test/localhost.crt", "./test/localhost.key")
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr,
		lgrpc.MustDialOption(lgrpc.WithTLS("./test/localhost.crt", "")),
	)
	if err != nil {
		t.Fatal(err)
	}
	c.PropagateContext = true
	testClient := NewTestClient(c.GRPC)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	res, err := testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	expectMsg := "Pong"
	if expectMsg != res.Msg {
		t.Errorf("expect msg to be %s, but got %s", expectMsg, res.Msg)
	}

	h.Drain()
}

func TestClientServerWithMutualTLS(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	h.ActivateMutualTLS(
		"./test/localhost.crt",
		"./test/localhost.key",
		"./test/localhost.crt",
	)
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr,
		lgrpc.MustDialOption(lgrpc.WithMutualTLS(
			"localhost",
			"./test/localhost.crt",
			"./test/localhost.key",
			"./test/localhost.crt",
		)),
	)
	if err != nil {
		t.Fatal(err)
	}
	c.PropagateContext = true
	testClient := NewTestClient(c.GRPC)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	res, err := testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	expectMsg := "Pong"
	if expectMsg != res.Msg {
		t.Errorf("expect msg to be %s, but got %s", expectMsg, res.Msg)
	}

	h.Drain()
}

func TestClientServer_WithOpenTracing(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c.AppendUnaryMiddleware(lgrpc.OpenTracingUnaryClientMiddleware())
	c.PropagateContext = true
	if err := c.WaitForStateReady(ctx); err != nil {
		t.Fatal(err)
	}
	testClient := NewTestClient(c.GRPC)

	ctx, _ = lcontext.NewTransitWithContext(ctx)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	span, ctx := tracing.StartSpanFromContext(ctx, "hello")
	defer span.Finish()

	res, err := testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	expectMsg := "Pong"
	if expectMsg != res.Msg {
		t.Errorf("expect msg to be %s, but got %s", expectMsg, res.Msg)
	}

	h.Drain()
}

func TestDrain(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx = config.TreeWithContext(ctx, tree)

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		t: tt,
	})
	addr := startServer(ctx, h)

	// Build client
	c, err := lgrpc.NewClient(ctx, addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c.PropagateContext = true
	testClient := NewTestClient(c.GRPC)

	// Prepare context
	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	// Start draining server
	h.Drain()

	_, err = testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err == nil {
		t.Fatal("expect to get an error when the server is drained")
	}
	if !strings.Contains(err.Error(), "grpc: the connection is unavailable") &&
		!strings.Contains(err.Error(), "transport is closing") &&
		!strings.Contains(err.Error(), "all SubConns are in TransientFailure") &&
		!strings.Contains(err.Error(), "rpc error: code = Unavailable") {
		t.Errorf("unexpected error %s", err)
	}
}

type MyTestServer struct {
	t *lt.T
}

func (s *MyTestServer) Hello(
	ctx context.Context, req *Request,
) (*Response, error) {
	log.Trace(ctx, "test.hello", "Calling Hello")

	expectLang := "en_GB"
	lang, ok := lcontext.Shipment(ctx, "lang").(string)
	if !ok || lang != expectLang {
		s.t.Errorf("expect lang %s from shipment, but got %s", expectLang, lang)
	}

	expectMsg := "Ping"
	if expectMsg != req.Msg {
		s.t.Errorf("expect to get %s, but got %s", expectMsg, req.Msg)
	}
	return &Response{Msg: "Pong"}, nil
}

func (s *MyTestServer) HelloFlow(
	stream Test_HelloFlowServer,
) error {
	log.Trace(stream.Context(), "test.hello", "Calling Hello")

	expectLang := "en_GB"
	lang, ok := lcontext.Shipment(stream.Context(), "lang").(string)
	if !ok || lang != expectLang {
		s.t.Errorf("expect lang %s from shipment, but got %s", expectLang, lang)
	}

	for {
		req, err := stream.Recv()
		switch err {
		case nil, io.EOF:
		default:
			s.t.Fatal(err)
		}
		if err == io.EOF {
			break
		}

		expectMsg := "Ping"
		if expectMsg != req.Msg {
			s.t.Errorf("expect to get %s, but got %s", expectMsg, req.Msg)
		}
	}

	if err := stream.Send(&Response{Msg: "Pong"}); err != nil {
		s.t.Fatal(err)
	}
	if err := stream.Send(&Response{Msg: "Pong"}); err != nil {
		s.t.Fatal(err)
	}
	return nil
}

func startServer(ctx context.Context, h *lgrpc.Server) string {
	addr := fmt.Sprintf("localhost:%d", lt.NextPort())

	// Start serving requests
	go func() {
		err := h.Serve(ctx, addr)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(50 * time.Millisecond)

	return addr
}
