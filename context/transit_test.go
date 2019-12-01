package context

import (
	"testing"
)

func TestTransit_ShortID(t *testing.T) {
	tests := []struct {
		in     string
		expect string
	}{
		{
			in:     "",
			expect: "",
		},
		{
			in:     "0",
			expect: "",
		},
		{
			in:     "01",
			expect: "",
		},
		{
			in:     "012",
			expect: "",
		},
		{
			in:     "0123",
			expect: "",
		},
		{
			in:     "01234",
			expect: "",
		},
		{
			in:     "012345",
			expect: "",
		},
		{
			in:     "0123456",
			expect: "",
		},
		{
			in:     "01234567",
			expect: "01234567",
		},
		{
			in:     "012345678",
			expect: "01234567",
		},
		{
			in:     "0123456789",
			expect: "01234567",
		},
	}

	for i, test := range tests {
		tr := transit{ID: test.in}
		got := tr.ShortID()
		if got != test.expect {
			t.Errorf("%d - expect Short to be equal %s, but got %s", i, test.expect, got)
		}
	}

}
