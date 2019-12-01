package config_test

import (
	"testing"

	"github.com/deixis/spine/config"
)

// TestDefaultAdapters tests whether the default adapters are registered
func TestDefaultAdapters(t *testing.T) {
	expected := []string{"consul", "file"}

	l := config.Adapters()
	if len(l) != len(expected) {
		t.Fatalf("expect to get %d registered adapters, but got %d", len(expected), len(l))
	}

	for i := range expected {
		if l[i] != expected[i] {
			t.Errorf("expect to get adapter %s, but got %s", expected[i], l[i])
		}
	}
}
