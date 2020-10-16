package stream

import (
	"context"

	"github.com/deixis/spine/contextutil"
)

type MsgHandler func(context.Context, []byte) error

type Stream interface {
	Pub
	Sub

	// Start does the initialisation work to bootstrap a Stream adapter.
	// For example, this function may open a connection, start an event loop, etc.
	Start(ctx context.Context) error
	// Drain signals to the Stream client/server that inbound messages should
	// no longer be accepted, but outbound messages can still be delivered.
	Drain()
	// Close closes the client/server for both inbound/outbound messages
	Close() error
}

// Pub is the publish interface
type Pub interface {
	// Publish publishes data to the channel ch
	// Publish(ctx context.Context, ch string, data []byte) error
	Publish(ctx context.Context, ch string, data []byte) error
}

// Sub is the subscribe interface
type Sub interface {
	// Subscribe subscribes the message handler h to the channel ch.
	// All subscriptions with the same q will form a queue group.
	// Each message will be delivered to only one subscriber per queue group.
	// Subscribe(q, ch string, h MsgHandler) error
	Subscribe(q, ch string, f MsgHandler, opts ...SubscriptionOption) (Subscription, error)
}

// Subscription represents a subscription to the streaming platform
type Subscription interface {
	// Unsubscribe removes interest in the subscription.
	// For durables, it means that the durable interest is also removed from
	// the server. Restarting a durable with the same name will not resume
	// the subscription, it will be considered a new one.
	Unsubscribe() error
}

type SubscriptionOption interface{}

type contextKey struct{}

var activePubContextKey = contextKey{}

// PubFromContext returns a `Pub` instance associated with `ctx`, or
// the local `Pub` if no instance could be found.
func PubFromContext(ctx contextutil.ValueContext) Pub {
	val := ctx.Value(activePubContextKey)
	if o, ok := val.(Pub); ok {
		return o
	}
	return NopStream()
}

// PubWithContext returns a copy of parent in which the `Pub` is stored
func PubWithContext(ctx context.Context, pub Pub) context.Context {
	return context.WithValue(ctx, activePubContextKey, pub)
}

var activeSubContextKey = contextKey{}

// SubFromContext returns a `Sub` instance associated with `ctx`, or
// the local `Sub` if no instance could be found.
func SubFromContext(ctx contextutil.ValueContext) Sub {
	val := ctx.Value(activeSubContextKey)
	if o, ok := val.(Sub); ok {
		return o
	}
	return NopStream()
}

// SubWithContext returns a copy of parent in which the `Sub` is stored
func SubWithContext(ctx context.Context, sub Sub) context.Context {
	return context.WithValue(ctx, activeSubContextKey, sub)
}

var activeStreamContextKey = contextKey{}

// FromContext returns a `Stream` instance associated with `ctx`, or
// the local `Sub` if no instance could be found.
func FromContext(ctx contextutil.ValueContext) Stream {
	val := ctx.Value(activeStreamContextKey)
	if o, ok := val.(Stream); ok {
		return o
	}
	return NopStream()
}

// WithContext returns a copy of parent in which `Stream`, `Pub`, and `Sub` are stored
func WithContext(ctx context.Context, ps Stream) context.Context {
	return PubWithContext(
		SubWithContext(
			context.WithValue(ctx, activeStreamContextKey, ps),
			ps,
		),
		ps,
	)
}
