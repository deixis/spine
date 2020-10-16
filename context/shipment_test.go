package context_test

import (
	"context"
	"testing"

	scontext "github.com/deixis/spine/context"
)

func TestShipment_Range(t *testing.T) {
	tests := []struct {
		key    string
		val    interface{}
		expect int
	}{
		{
			key:    "a",
			val:    "aval",
			expect: 1,
		},
		{
			key:    "b",
			val:    "bval",
			expect: 2,
		},
		{
			key:    "c",
			val:    "cval",
			expect: 3,
		},
	}

	ctx := context.Background()
	for i, test := range tests {
		ctx = scontext.WithShipment(ctx, test.key, test.val)

		var got int
		scontext.ShipmentRange(ctx, func(k string, value interface{}) bool {
			got++
			return true
		})

		if got != test.expect {
			t.Errorf("%d - expect to get %d shipments, but got %d", i, test.expect, got)
		}
	}

}
