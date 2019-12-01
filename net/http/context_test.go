package http_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/deixis/spine/config"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net/http"
	lt "github.com/deixis/spine/testing"
)

type Foo struct {
	Label     string
	Threshold int
}

// TestDefaultBehaviour creates an HTTP endpoint and send a request from the client
// It ensures the context is NOT propagated upstream
func TestDefaultBehaviour(t *testing.T) {
	tt := lt.New(t)
	appCtx, _ := tt.WithCancel(context.Background())

	// Build handler
	h := http.NewServer()
	defer h.Drain()
	var gotContext context.Context
	h.HandleFunc("/test", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		log.Trace(ctx, "http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx, cancel := context.WithCancel(appCtx)
	defer cancel()
	ctx, tr := lcontext.NewTransitWithContext(ctx)

	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	// Send request
	client := http.Client{}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if tr.UUID() == lcontext.TransitFromContext(gotContext).UUID() {
		t.Error("expect contexts to be different")
	}
	lcontext.ShipmentRange(ctx, func(key string, expect interface{}) bool {
		v := lcontext.Shipment(gotContext, key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestAllowContext creates an HTTP endpoint and send a request from the client
// It ensures the context is NOT propagated upstream by the client by default
func TestAllowContext(t *testing.T) {
	tt := lt.New(t)
	tt.Config().Request.AllowContext = true
	appCtx, _ := tt.WithCancel(context.Background())

	// Build handler
	h := http.NewServer()
	var gotContext context.Context
	h.HandleFunc("/test", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		log.Trace(ctx, "http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx, cancel := context.WithCancel(appCtx)
	defer cancel()
	ctx, tr := lcontext.NewTransitWithContext(ctx)

	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	// Send request
	client := http.Client{}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if tr.UUID() == lcontext.TransitFromContext(gotContext).UUID() {
		t.Error("expect contexts to be different")
	}
	lcontext.ShipmentRange(ctx, func(key string, expect interface{}) bool {
		v := lcontext.Shipment(gotContext, key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestAllowContext creates an HTTP endpoint and send a request from the client
// It ensures the context is propagated, but blocked on the upstream node
func TestBlockContext(t *testing.T) {
	tt := lt.New(t)
	appCtx, _ := tt.WithCancel(context.Background())

	// Build handler
	h := http.NewServer()
	var gotContext context.Context
	h.HandleFunc("/test", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		log.Trace(ctx, "http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx, cancel := context.WithCancel(appCtx)
	defer cancel()
	ctx, tr := lcontext.NewTransitWithContext(ctx)

	log.Trace(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	// Send request
	client := http.Client{
		PropagateContext: true,
	}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if tr.UUID() == lcontext.TransitFromContext(gotContext).UUID() {
		t.Error("expect contexts to be different")
	}
	lcontext.ShipmentRange(ctx, func(key string, expect interface{}) bool {
		v := lcontext.Shipment(gotContext, key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestPropagateContext creates an HTTP endpoint and send a request from the client
// It ensures the context is propagated and accepted upstream
func TestPropagateContext(t *testing.T) {
	tt := lt.New(t)
	appCtx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	appCtx = config.TreeWithContext(appCtx, tree)

	// Build handler
	h := http.NewServer()
	var gotContext context.Context
	h.HandleFunc("/test", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		log.Trace(ctx, "http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx, _ := lcontext.NewTransitWithContext(appCtx)
	log.Trace(ctx, "prepare", "Prepare context")
	lcontext.WithShipment(ctx, "lang", "en_GB")
	lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	lcontext.WithShipment(ctx, "flag", 3)

	// Send request
	client := http.Client{
		PropagateContext: true,
	}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	expectTransit := lcontext.TransitFromContext(ctx)
	gotTransit := lcontext.TransitFromContext(gotContext)
	if expectTransit.UUID() != gotTransit.UUID() {
		t.Errorf("expect context to have UUID %s, but got %s", expectTransit.UUID(), gotTransit.UUID())
	}
	if gotContext == nil {
		t.Fatalf("expect KV to not be nil")
	}
	lcontext.ShipmentRange(ctx, func(key string, expect interface{}) bool {
		got := lcontext.Shipment(gotContext, key)
		if expect != got {
			t.Errorf("expect to value for key %s to be %v, but got %v", key, expect, got)
		}
		return false
	})
}

func TestShipments(t *testing.T) {
	tt := lt.New(t)
	appCtx, _ := tt.WithCancel(context.Background())
	tree, err := config.TreeFromMap(map[string]interface{}{
		"request": map[string]interface{}{
			"allow_context": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	appCtx = config.TreeWithContext(appCtx, tree)

	// Build handler
	h := http.NewServer()
	h.HandleFunc("/test", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		log.Trace(ctx, "http.test", "Test endpoint called")

		lang, ok := lcontext.Shipment(ctx, "lang").(string)
		if !ok || lang != "en_GB" {
			t.Errorf("expect to get lang en_GB, but got %s", lang)
		}
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx, _ := lcontext.NewTransitWithContext(appCtx)
	ctx = lcontext.WithShipment(ctx, "prepare", "Prepare context")
	ctx = lcontext.WithShipment(ctx, "lang", "en_GB")
	ctx = lcontext.WithShipment(ctx, "ip", "10.0.0.21")
	ctx = lcontext.WithShipment(ctx, "flag", 3)

	// Send request
	client := http.Client{
		PropagateContext: true,
	}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}
}

func startServer(ctx context.Context, h *http.Server) string {
	addr := fmt.Sprintf("127.0.0.1:%d", lt.NextPort())
	h.HandleFunc("/preflight", http.GET, func(
		ctx context.Context, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})

	// Start serving requests
	go func() {
		err := h.Serve(ctx, addr)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	for attempt := 1; attempt <= 10; attempt++ {
		res, err := http.Get(ctx, fmt.Sprintf("http://%s/preflight", addr))
		if err == nil && res.StatusCode == http.StatusOK {
			break
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	return addr
}
