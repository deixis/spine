package net_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deixis/spine/net"
	lt "github.com/deixis/spine/testing"
)

// TestHandlerRegistration tests whether a handler that listen on a given
// address can be registered once and only once
func TestHandlerRegistration(t *testing.T) {
	tt := lt.New(t)
	reg := net.NewReg(tt.Logger())
	addr := "localhost:8080"
	h := NewDummyH()

	// Register a first time
	if p, msg := lt.DidPanic(func() { reg.Add(addr, h) }); p {
		t.Error("expect to be able to add handler", msg)
	}

	// Attempt to register a second time
	if p, _ := lt.DidPanic(func() { reg.Add(addr, h) }); !p {
		t.Error("expect to fail when the same handler has already been registered")
	}

	// Attempt to register another handler on the same address
	if p, _ := lt.DidPanic(func() { reg.Add(addr, NewDummyH()) }); !p {
		t.Error("expect to fail when another handler has already been registered on the same address")
	}
}

// TestServe tests whether all handlers are correctly started
func TestServe(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	reg := net.NewReg(tt.Logger())

	l := []struct {
		h    *dummyH
		addr string
	}{
		{h: NewDummyH(), addr: fmt.Sprintf("localhost:%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf("localhost:%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
	}

	// Register handlers
	for i, item := range l {
		if p, msg := lt.DidPanic(func() { reg.Add(item.addr, item.h) }); p {
			t.Error("expect to be able to add handler", msg, i)
		}
	}

	// Start serving
	if err := reg.Serve(ctx); err != nil {
		t.Error("expect Serve to not return an error", err)
	}

	// Serve ensures that handlers are booting, but they might not run (yet)
	time.Sleep(256 * time.Microsecond)

	// Ensure all handlers have been started
	for i, item := range l {
		if !item.h.IsRunning() {
			t.Error("expect handler to be running", i, item.addr)
		}
	}
}

// TestServeEmptyRegistry tests whether Serve returns an error when the registry
// is empty
func TestServeEmptyRegistry(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	reg := net.NewReg(tt.Logger())

	if err := reg.Serve(ctx); err != net.ErrEmptyReg {
		t.Error("expect Serve to return an error when the registry is empty", err)
	}
}

// TestDrain tests whether all handlers are properly drained
func TestDrain(t *testing.T) {
	tt := lt.New(t)
	ctx, _ := tt.WithCancel(context.Background())
	reg := net.NewReg(tt.Logger())

	l := []struct {
		h    *dummyH
		addr string
	}{
		{h: NewDummyH(), addr: fmt.Sprintf("localhost:%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf("localhost:%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
		{h: NewDummyH(), addr: fmt.Sprintf(":%d", lt.NextPort())},
	}

	// Register handlers
	for i, item := range l {
		if p, msg := lt.DidPanic(func() { reg.Add(item.addr, item.h) }); p {
			t.Errorf("expect to be able to add handler (%d - %s)", i, msg)
		}
	}

	// Start serving
	if err := reg.Serve(ctx); err != nil {
		t.Error("expect Serve to not return an error", err)
	}

	// Start draining
	reg.Drain()

	// Ensure all handlers have been started
	for i, item := range l {
		if item.h.IsRunning() {
			t.Error("expect handler to have been stopped", i, item.addr)
		}
	}
}

type dummyH struct {
	mu sync.Mutex

	Running bool
	Stop    chan struct{}
	Done    chan struct{}
}

func NewDummyH() *dummyH {
	return &dummyH{
		Stop: make(chan struct{}, 1),
		Done: make(chan struct{}, 1),
	}
}

func (h *dummyH) Serve(ctx context.Context, addr string) error {
	h.Run(true)
	<-h.Stop
	h.Run(false)
	h.Done <- struct{}{}
	return nil
}

func (h *dummyH) Drain() {
	time.Sleep(256 * time.Microsecond)
	h.Stop <- struct{}{}
	<-h.Done
}

func (h *dummyH) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Running
}

func (h *dummyH) Run(f bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Running = f
}
