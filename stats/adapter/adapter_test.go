package adapter_test

import (
	"testing"

	"github.com/deixis/spine/stats/adapter"
)

// TestDefaultAdapters tests whether the default adapters are registered
func TestDefaultAdapters(t *testing.T) {
	expected := []string{"statsd"}

	l := adapter.Adapters()
	if len(l) != len(expected) {
		t.Fatalf("expect to get %d registered adapters, but got %d", len(expected), len(l))
	}

	for i := range expected {
		if l[i] != expected[i] {
			t.Errorf("expect to get adapter %s, but got %s", expected[i], l[i])
		}
	}
}
