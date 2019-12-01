// Package consul is a wrapper around the Hashicorp Consul service discovery functionnality
//
// Consul is a highly available and distributed service discovery and key-value store designed
// with support for the modern data center to make distributed systems and configuration easy.
package consul

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/deixis/spine/config"
	"github.com/deixis/spine/disco"
	"github.com/deixis/spine/log"
)

// Name contains the adapter registered name
const Name = "consul"

// New returns a new file config store
func New(tree config.Tree) (disco.Agent, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, err
	}

	// Configure client
	cc := api.DefaultConfig()
	cc.Address = config.Address
	cc.Datacenter = config.DC
	cc.Token = config.Token

	// Build Consul client
	consul, err := api.NewClient(cc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialise Consul client")
	}

	return &Agent{
		consul:       consul,
		consulConfig: cc,
		config:       config,
		serviceIDs:   map[string]struct{}{},
	}, nil
}

// Config contains the configuration to start a service discovery agent
type Config struct {
	AdvertiseAddr string   `json:"advertise_addr"`
	DefaultTags   []string `json:"default_tags"`
	Address       string   `json:"address"`
	DC            string   `json:"dc"`
	Token         string   `json:"token"`
}

type Agent struct {
	mu sync.Mutex

	consul       *api.Client
	consulConfig *api.Config
	config       *Config
	advertAddr   string

	// serviceIDs caches the list of services registered
	serviceIDs map[string]struct{}
}

func (a *Agent) Register(ctx context.Context, r *disco.Registration) (string, error) {
	tags := append(a.config.DefaultTags, r.Tags...)
	reg := api.AgentServiceRegistration{
		ID:      r.ID,
		Name:    r.Name,
		Port:    int(r.Port),
		Address: r.Addr,
		Tags:    tags,
	}
	if reg.ID == "" {
		reg.ID = uuid.New().String()
	}
	if a.advertAddr != "" {
		if net.ParseIP(a.advertAddr) != nil {
			reg.Address = a.advertAddr
		} else {
			addr, err := lookupHost(ctx, a.advertAddr)
			if err != nil {
				log.FromContext(ctx).Warning(
					"disco.register.invalid_ad_addr",
					"Advertise addr is not an IP, nor a valid DNS A record",
					log.String("advertise_address", a.advertAddr),
				)
			}
			reg.Address = addr
		}
	}

	log.FromContext(ctx).Trace("disco.register", "Register service",
		log.String("adapter", "consul"),
		log.String("service_name", reg.Name),
		log.String("instance_id", reg.ID),
		log.String("instance_address", reg.Address),
		log.Int("instance_port", reg.Port),
		log.String("instance_tags", strings.Join(reg.Tags, ", ")),
	)
	err := a.consul.Agent().ServiceRegister(&reg)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.serviceIDs[reg.ID] = struct{}{}
	a.mu.Unlock()
	return reg.ID, nil
}

func (a *Agent) Deregister(ctx context.Context, id string) error {
	log.FromContext(ctx).Trace("disco.deregister", "Deregister service",
		log.String("id", id),
		log.String("adapter", "consul"),
	)

	err := a.consul.Agent().ServiceDeregister(id)
	if err != nil {
		return err
	}

	a.mu.Lock()
	delete(a.serviceIDs, id)
	a.mu.Unlock()
	return nil
}

func (a *Agent) Services(
	ctx context.Context, tags ...string,
) (map[string]disco.Service, error) {
	r, err := a.consul.Agent().Services()
	if err != nil {
		return nil, err
	}

	svcs := map[string]disco.Service{}
	for id, s := range r {
		if !isSubset(s.Tags, tags) {
			continue
		}

		v, ok := svcs[s.Service]
		if !ok {
			v = &service{
				name: s.Service,
				watch: func() disco.Watcher {
					return a.watch(ctx, s.Service, tags...)
				},
			}
			svcs[s.Service] = v
		}

		svc := v.(*service)
		svc.instances = append(svc.instances, &disco.Instance{
			ID:   id,
			Name: s.Service,
			Host: s.Address,
			Port: uint16(s.Port),
			Tags: s.Tags,
		})
	}
	return svcs, nil
}

func (a *Agent) Service(
	ctx context.Context, name string, tags ...string,
) (disco.Service, error) {
	q := a.buildQueryOptions()
	instances, _, err := a.service(ctx, name, q, tags...)
	if err != nil {
		return nil, err
	}
	return &service{
		name:      name,
		instances: instances,
		watch: func() disco.Watcher {
			return a.watch(ctx, name, tags...)
		},
	}, nil
}

func (a *Agent) Leave(ctx context.Context) {
	for id := range a.serviceIDs {
		err := a.Deregister(ctx, id)
		if err != nil {
			log.FromContext(ctx).Warning(
				"disco.leave.failure", "Could not de-register service",
				log.String("service_id", id),
			)
		}
	}

	a.mu.Lock()
	a.serviceIDs = map[string]struct{}{}
	a.mu.Unlock()
}

func (a *Agent) watch(
	ctx context.Context, name string, tags ...string,
) disco.Watcher {
	var waitIndex uint64
	next := func(ctx context.Context) ([]*disco.Instance, error) {
		q := a.buildQueryOptions()
		q.WaitIndex = waitIndex
		instances, meta, err := a.service(ctx, name, q, tags...)
		if err != nil {
			return nil, err
		}
		waitIndex = meta.LastIndex
		return instances, nil
	}
	return newWatcher(ctx, next)
}

func (a *Agent) service(
	ctx context.Context, name string, q *api.QueryOptions, tags ...string,
) ([]*disco.Instance, *api.QueryMeta, error) {
	var tag string
	if len(tags) > 0 {
		tag = tags[0]
	}

	r, meta, err := a.consul.Health().Service(name, tag, true, q)
	if err != nil {
		return nil, nil, err
	}

	var instances []*disco.Instance
	for _, chk := range r {
		if !isSubset(chk.Service.Tags, tags) {
			continue
		}

		_, local := a.serviceIDs[chk.Service.ID]
		instances = append(instances, &disco.Instance{
			Local: local,
			ID:    chk.Service.ID,
			Name:  chk.Service.Service,
			Host:  chk.Service.Address,
			Port:  uint16(chk.Service.Port),
			Tags:  chk.Service.Tags,
		})
	}
	return instances, meta, nil
}

func (a *Agent) buildQueryOptions() *api.QueryOptions {
	return &api.QueryOptions{
		Datacenter: a.consulConfig.Datacenter,
		Token:      a.consulConfig.Token,
	}
}

// service implements disco.Service
type service struct {
	name      string
	instances []*disco.Instance
	watch     func() disco.Watcher
}

func (s *service) Name() string {
	return s.name
}

func (s *service) Watch() disco.Watcher {
	return s.watch()
}

func (s *service) Instances() []*disco.Instance {
	return s.instances
}

type watcher struct {
	next   func(context.Context) ([]*disco.Instance, error)
	ctx    context.Context
	cancel func()

	diff disco.Diff
}

func newWatcher(
	ctx context.Context, n func(context.Context) ([]*disco.Instance, error),
) *watcher {
	child, cancelFunc := context.WithCancel(ctx)
	return &watcher{
		next:   n,
		ctx:    child,
		cancel: cancelFunc,
	}
}

func (w *watcher) Next() ([]*disco.Event, error) {
	if w.ctx.Err() != nil {
		return nil, disco.ErrWatcherClosed
	}

	instances, err := w.next(w.ctx)
	if err != nil {
		return nil, err
	}
	return w.diff.Apply(instances), nil
}

func (w *watcher) Close() error {
	w.cancel()
	return nil
}

// isSubset returns whether b is a subset of a
func isSubset(a, b []string) bool {
	if len(a) < len(b) {
		return false
	}
	if len(b) == 0 {
		return true
	}

	sort.Strings(a)
	sort.Strings(b)
	var matches int
	for i := 0; i < len(a); i++ {
		if a[i] == b[matches] {
			matches++
		}
		if matches == len(b) {
			return true
		}
	}
	return false
}

func lookupHost(ctx context.Context, host string) (string, error) {
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		a, ok := formatIP(a)
		if !ok {
			continue
		}
		return a, nil
	}
	return "", errors.New("error parsing all addresses")
}

// formatIP returns ok = false if addr is not a valid textual representation of an IP address.
// If addr is an IPv4 address, return the addr and ok = true.
// If addr is an IPv6 address, return the addr enclosed in square brackets and ok = true.
func formatIP(addr string) (addrIP string, ok bool) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return addr, true
	}
	return "[" + addr + "]", true
}
