package stream

import (
	"context"
	"errors"
)

// nopStream is a Stream adapter which does not do anything.
type nopStream struct{}

// NopStream returns a Stream adapter which discards all pubs
func NopStream() Stream {
	return &nopStream{}
}

func (ps *nopStream) Start(ctx context.Context) error {
	return nil
}

func (ps *nopStream) Publish(ctx context.Context, ch string, data []byte) error {
	return nil
}

func (ps *nopStream) Subscribe(q, ch string, f MsgHandler, opts ...SubscriptionOption) (Subscription, error) {
	return nil, errors.New("cannot subscribe to nop stream")
}

func (ps *nopStream) Drain() {
}

func (ps *nopStream) Close() error {
	return nil
}
