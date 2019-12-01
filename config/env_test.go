package config_test

import (
	"os"
	"testing"

	"github.com/deixis/spine/config"
)

func TestValueOf(t *testing.T) {
	os.Setenv("SPINE_TEST_CONFIG_VALUEOF", "yay")

	tests := []struct {
		in  string
		out string
	}{
		{in: "foo", out: "foo"},
		{in: "$SPINE_TEST_CONFIG_VALUEOF", out: "yay"},
		{in: "$DOES_NOT_EXIST_ABCDEFG0123459", out: ""},
		{in: "$", out: "$"},
	}

	for _, test := range tests {
		res := config.ValueOf(test.in)
		if res != test.out {
			t.Errorf("expect ValueOf to return %s, but got %s", test.out, res)
		}
	}
}
