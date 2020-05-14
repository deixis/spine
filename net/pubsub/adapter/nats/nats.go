package nats

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/deixis/spine/config"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net"
	"github.com/deixis/spine/net/pubsub"
	pb "github.com/deixis/spine/net/pubsub/adapter/nats/natspb"
	"github.com/deixis/spine/tracing"
	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

const (
	Name = "nats"
)

type NATS struct {
	mu    sync.RWMutex
	state uint32

	rootctx context.Context
	log     log.Logger
	conn    *nats.Conn
	subs    []*nats.Subscription

	Config Config
}

type Config struct {
	URI  string `toml:"uri"`
	Name string `toml:"name"`
}

func New(tree config.Tree) (pubsub.PubSub, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal pubsub.nats config")
	}

	return &NATS{
		state:  net.StateUp,
		Config: *config,
	}, nil
}

func (ps *NATS) Start(ctx context.Context) error {
	ps.rootctx = ctx
	ps.log = log.FromContext(ctx)

	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(ps.Config.Name),
		nats.DisconnectHandler(func(conn *nats.Conn) {
			ps.log.Warning("pubsub.disconnect", "Lost connection to pubsub",
				log.Uint64("reconnects", conn.Stats().Reconnects),
			)
		}),
		nats.ErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
			ps.log.Error(
				"pubsub.error",
				"Error encountered while processing inbound messages",
				log.Uint64("reconnects", conn.Stats().Reconnects),
				log.Error(err),
			)
		}),
	}

	conn, err := nats.Connect(ps.Config.URI, opts...)
	if err != nil {
		return errors.Wrap(err, "cannot connect to NATS")
	}
	ps.conn = conn
	return nil
}

func (ps *NATS) Publish(ctx context.Context, ch string, data []byte) error {
	pb := pb.Message{
		Payload: data,
	}

	tr := lcontext.TransitFromContext(ctx)
	if tr != nil {
		data, err := tr.MarshalBinary()
		if err != nil {
			return errors.Wrap(err, "error marshalling transit")
		}
		pb.Transit = data
	}

	data, err := proto.Marshal(&pb)
	if err != nil {
		return errors.Wrap(err, "cannot marshal NATS message")
	}
	return ps.conn.Publish(ch, data)
}

func (ps *NATS) Subscribe(q, ch string, f pubsub.MsgHandler) error {
	if !ps.isState(net.StateUp) {
		log.Warn(ps.rootctx, "pubsub.pub.draining", "PubSub is down or draining")
		return net.ErrDraining
	}

	sub, err := ps.conn.QueueSubscribe(ch, q, func(msg *nats.Msg) {
		pb := pb.Message{}
		if err := proto.Unmarshal(msg.Data, &pb); err != nil {
			ps.log.Error("pubsub.unmarshal.err", "Cannot unmarshal message",
				log.Error(err),
			)
			return
		}

		// Create request root context
		ctx, cancel := context.WithCancel(ps.rootctx)
		defer cancel()

		// Extract transit
		var tr lcontext.Transit
		if len(pb.Transit) > 0 {
			tr = lcontext.TransitFactory()
			err := tr.UnmarshalBinary(pb.Transit)
			if err != nil {
				log.Err(ctx, "pubsub.transit.err", "Error unmarshalling transit",
					log.Error(err),
				)
				ctx, tr = lcontext.NewTransitWithContext(ctx)
			} else {
				ctx = lcontext.TransitWithContext(ctx, tr)
			}
		} else {
			ctx, tr = lcontext.NewTransitWithContext(ctx)
		}

		// TODO: Create follow up transit

		// Attach transit-specific services
		ctx = lcontext.WithTracer(ctx, tracing.FromContext(ctx))
		ctx = lcontext.WithLogger(ctx, log.FromContext(ctx))

		// Call handler
		f(ctx, pb.Payload)
	})
	if err != nil {
		return errors.Wrap(err, "error subscribing to NATS")
	}

	ps.mu.Lock()
	ps.subs = append(ps.subs, sub)
	ps.mu.Unlock()
	return nil
}

func (ps *NATS) Drain() {
	atomic.StoreUint32(&ps.state, net.StateDrain)
	ps.mu.Lock()
	for _, sub := range ps.subs {
		sub.Unsubscribe()
	}
	ps.mu.Unlock()
	atomic.StoreUint32(&ps.state, net.StateDown)
}

func (ps *NATS) Close() error {
	atomic.StoreUint32(&ps.state, net.StateDown)
	if ps.conn.IsClosed() {
		return nil
	}
	ps.conn.Flush()
	ps.conn.Close()
	return nil
}

// isState checks the current server state
func (ps *NATS) isState(state uint32) bool {
	return atomic.LoadUint32(&ps.state) == uint32(state)
}
