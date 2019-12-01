// Package stan is an adapter for NATS Streaming System
package stan

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/pkg/errors"
	"github.com/deixis/spine/config"
	lcontext "github.com/deixis/spine/context"
	"github.com/deixis/spine/log"
	"github.com/deixis/spine/net"
	"github.com/deixis/spine/net/stream"
	pb "github.com/deixis/spine/net/stream/adapter/stan/stanpb"
	"github.com/deixis/spine/tracing"
)

const (
	Name = "stan"
)

type Stan struct {
	mu    sync.RWMutex
	state uint32

	rootctx context.Context
	log     log.Logger
	conn    stan.Conn
	subs    []stan.Subscription

	Config Config
}

type Config struct {
	// URI is the NATS resource identifier
	URI string `toml:"uri"`
	// ClusterID is the Stan cluster unique identifier
	ClusterID string `toml:"cluster_id"`
	// ClientID can contain only alphanumeric and `-` or `_` characters.
	ClientID string `toml:"client_id"`
	// Subscription contains the default subscription config
	Subscription SubscriptionConfig `toml:"subscription"`
}

type SubscriptionConfig struct {
	// DurableName, if set will survive client restarts.
	DurableName string `toml:"durable_name"`
	// Controls the number of messages the cluster will have inflight without an ACK.
	MaxInflight int `toml:"max_inflight"`
	// Controls the time the cluster will wait for an ACK for a given message.
	AckWait time.Duration `toml:"ack_wait"`
}

func New(tree config.Tree) (stream.Stream, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal stream.Stan config")
	}

	return &Stan{
		state:  net.StateUp,
		Config: *config,
	}, nil
}

func (s *Stan) Start(ctx context.Context) error {
	s.rootctx = ctx
	s.log = log.FromContext(ctx)

	nc, err := nats.Connect(s.Config.URI, []nats.Option{
		nats.Name(s.Config.ClientID),
		nats.MaxReconnects(-1),
		nats.ReconnectBufSize(-1),
	}...)
	if err != nil {
		return errors.Wrap(err, "cannot connect to NATS server")
	}

	opts := []stan.Option{
		stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(conn stan.Conn, reason error) {
			s.log.Warning("stream.disconnect", "Lost connection to stream",
				log.Uint64("reconnects", conn.NatsConn().Stats().Reconnects),
				log.Error(reason),
			)
		}),
	}

	conn, err := stan.Connect(s.Config.ClusterID, s.Config.ClientID, opts...)
	if err != nil {
		return errors.Wrap(err, "cannot connect to NATS streaming server (clusterID ?).")
	}
	s.conn = conn
	return nil
}

func (s *Stan) Publish(ctx context.Context, ch string, data []byte) error {
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
		return errors.Wrap(err, "cannot marshal Stan message")
	}

	if err := s.conn.Publish(ch, data); err != nil {
		return err
	}
	return nil
}

func (s *Stan) Subscribe(
	q, ch string, f stream.MsgHandler, opts ...stream.SubscriptionOption,
) (stream.Subscription, error) {
	if !s.isState(net.StateUp) {
		log.Warn(s.rootctx, "stream.pub.draining", "Stream is down or draining")
		return nil, net.ErrDraining
	}

	defaultOpts := []stan.SubscriptionOption{
		stan.SetManualAckMode(),
	}

	// Default options
	sConfig := s.Config.Subscription
	if len(sConfig.DurableName) > 0 {
		defaultOpts = append(defaultOpts, stan.DurableName(sConfig.DurableName))
	}
	if sConfig.MaxInflight > 0 {
		defaultOpts = append(defaultOpts, stan.MaxInflight(sConfig.MaxInflight))
	}
	if sConfig.AckWait > 0 {
		defaultOpts = append(defaultOpts, stan.AckWait(sConfig.AckWait))
	}

	// Subscription options
	sub, err := s.conn.QueueSubscribe(ch, q, func(msg *stan.Msg) {
		pb := pb.Message{}
		if err := proto.Unmarshal(msg.Data, &pb); err != nil {
			s.log.Error("stream.unmarshal.err", "Cannot unmarshal message",
				log.Error(err),
			)
			return
		}

		// Create request root context
		ctx, cancel := context.WithCancel(s.rootctx)
		defer cancel()

		// Extract transit
		var tr lcontext.Transit
		if len(pb.Transit) > 0 {
			tr = lcontext.TransitFactory()
			err := tr.UnmarshalBinary(pb.Transit)
			if err != nil {
				log.Err(ctx, "stream.transit.err", "Error unmarshalling transit",
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
		if err := f(ctx, pb.Payload); err != nil {
			log.Warn(ctx, "stream.proc.err", "Error processing msg",
				log.Error(err),
			)
			return
		}

		if err := msg.Ack(); err != nil {
			log.Warn(ctx, "stream.ack.err", "Error acknowledging msg",
				log.Error(err),
			)
		}
	}, defaultOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error subscribing to NATS")
	}

	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()
	return sub, nil
}

func (s *Stan) Drain() {
	atomic.StoreUint32(&s.state, net.StateDrain)
	s.mu.Lock()
	for _, sub := range s.subs {
		// Close removes this subscriber from the server, but unlike Unsubscribe(),
		// the durable interest is not removed. If the client has connected to a server
		// for which this feature is not available, Close() will return a ErrNoServerSupport
		// error.
		if err := sub.Close(); err != nil {
			log.Warn(s.rootctx, "stream.stan.drain.err", "Error closing subscription",
				log.Error(err),
			)
		}
	}
	s.mu.Unlock()
	atomic.StoreUint32(&s.state, net.StateDown)
}

func (s *Stan) Close() error {
	atomic.StoreUint32(&s.state, net.StateDown)

	// If there are active subscriptions at the time of the close, they are implicitly closed
	// (not unsubscribed) by the cluster. This means that durable subscriptions are maintained.
	//
	// The wait on asynchronous publish calls are canceled and ErrConnectionClosed will be
	// reported to the registered AckHandler. It is possible that the cluster received and
	// persisted these messages.
	if err := s.conn.Close(); err != nil {
		log.Warn(s.rootctx, "stream.stan.close.err", "Error closing NATS streaming connection",
			log.Error(err),
		)
		return err
	}

	// Closing connection to NATS server
	s.conn.NatsConn().Close()
	return nil
}

// isState checks the current server state
func (s *Stan) isState(state uint32) bool {
	return atomic.LoadUint32(&s.state) == uint32(state)
}

type MsgHandler interface{}
