package local_test

import (
	"context"
	"testing"

	"github.com/deixis/spine/cache/adapter/local"
	"github.com/deixis/spine/config"
)

func TestCache(t *testing.T) {
	ctx := context.Background()

	cache, err := local.New(config.NopTree())
	if err != nil {
		t.Fatal(err)
	}

	// Create group which can hold 2 keys
	expect := []byte("bar")
	var load int
	group := cache.NewGroup("foo", 6, func(ctx context.Context, key string) ([]byte, error) {
		load++
		return expect, nil
	})

	// Store first
	got, err := group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 1 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Store second
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 2 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Store third
	got, err = group.Get(ctx, "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 3 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Ensure the second is still in the cache
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 3 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Ensure the first has been evicted
	got, err = group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 4 {
		t.Errorf("Expect to load data once, but got %d", load)
	}
}
