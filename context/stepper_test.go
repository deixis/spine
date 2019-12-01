package context

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

// TestInc tests whether the last counter is properly incremented
func TestInc(t *testing.T) {
	s := newStepper()

	res := s.Inc()
	expect := uint(1)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}

	res = s.Inc()
	expect = uint(2)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}

	s = s.Child()

	res = s.Inc()
	expect = uint(1)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}
}

// TestStringTestInc tests the string representation of a stepper
func TestString(t *testing.T) {
	tests := []struct {
		in     *stepper
		expect string
	}{
		{
			in:     newStepper(),
			expect: "0000",
		},
		{
			in: &stepper{
				Steps: []uint32{20},
				I:     0,
			},
			expect: "0020",
		},
		{
			in: &stepper{
				Steps: []uint32{10, 100, 1000},
				I:     2,
			},
			expect: "0010_0100_1000",
		},
	}

	for i, test := range tests {
		got := test.in.String()
		if got != test.expect {
			t.Errorf("%d - expect String to be equal %s, but got %s", i, test.expect, got)
		}
	}

}

func TestStepper_Marshal(t *testing.T) {
	s := newStepper()
	s.Inc()
	s.Inc()

	s = s.Child()
	s.Inc()

	expect := *s             // Copy stepper
	var network bytes.Buffer // Stand-in for the network.

	// Create an encoder and send a value.
	enc := gob.NewEncoder(&network)
	if err := enc.Encode(s); err != nil {
		t.Fatal("encode:", err)
	}

	// Create a decoder and receive a value.
	s = &stepper{}
	dec := gob.NewDecoder(&network)
	if err := dec.Decode(s); err != nil {
		t.Fatal("encode:", err)
	}

	if expect.I != s.I {
		t.Errorf("expect I %d, but got %d", expect.I, s.I)
	}
	if !reflect.DeepEqual(expect.Steps, s.Steps) {
		t.Errorf("expect I %v, but got %v", expect.Steps, s.Steps)
	}
}
