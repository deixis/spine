package pubsub

import "context"

// nopPubSub is a PubSub adapter which does not do anything.
type nopPubSub struct{}

// NopPubSub returns a PubSub adapter which discards all pubs
func NopPubSub() PubSub {
	return &nopPubSub{}
}

func (ps *nopPubSub) Start(ctx context.Context) error {
	return nil
}

func (ps *nopPubSub) Publish(ctx context.Context, ch string, data []byte) error {
	return nil
}

func (ps *nopPubSub) Subscribe(q, ch string, h MsgHandler) error {
	return nil
}

func (ps *nopPubSub) Drain() {
}

func (ps *nopPubSub) Close() error {
	return nil
}
