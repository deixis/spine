package consul_test

import (
	"net/url"
	"testing"

	"github.com/deixis/spine/config/adapter/consul"
)

func TestConsul(t *testing.T) {
	tests := []struct {
		in    string
		addr  string
		key   string
		dc    string
		token string
		err   error
	}{
		{in: "consul://localhost:32773/config/spine.json", addr: "localhost:32773", key: "config/spine.json"},
		{in: "consul://localhost/config/spine.json", addr: "localhost", key: "config/spine.json"},
		{in: "consul://localhost/config/spine.json?dc=dc1&token=123", addr: "localhost", key: "config/spine.json", dc: "dc1", token: "123"},
		{in: "consul://localhost/", err: consul.ErrMissingStoreKey},
	}

	for _, test := range tests {
		uri, err := url.Parse(test.in)
		if err != nil {
			t.Errorf("expect to receive a valid url, but got %s", test.in)
		}

		res, err := consul.New(uri)
		if err != test.err {
			t.Errorf("expect err to return %v, but got %v", test.err, err)
		}
		if err != nil {
			continue
		}

		store, ok := res.(*consul.Store)
		if !ok {
			t.Fatalf("expect to receive a *consul.Store, but got %v", store)
		}
		if store.Key != test.key {
			t.Errorf("expect key to return %s, but got %s", test.key, store.Key)
		}
		if store.Config.Address != test.addr {
			t.Errorf("expect key to return %s, but got %s", test.addr, store.Config.Address)
		}
		if store.Config.Datacenter != test.dc {
			t.Errorf("expect datacenter to return %s, but got %s", test.dc, store.Config.Datacenter)
		}
		if store.Config.Token != test.token {
			t.Errorf("expect datacenter to return %s, but got %s", test.dc, store.Config.Datacenter)
		}
	}
}
