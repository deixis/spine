package disco_test

import (
	"reflect"
	"testing"

	"github.com/deixis/spine/disco"
)

func TestDiff(t *testing.T) {
	// Add first node
	diff := disco.Diff{}
	a := []*disco.Instance{
		&disco.Instance{
			ID:   "alpha",
			Name: "Instance Alpha",
			Host: "127.0.0.1",
			Port: 1001,
		},
	}
	expect := []*disco.Event{
		&disco.Event{
			Op:       disco.Add,
			Instance: a[0],
		},
	}

	res := diff.Apply(a)
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("expect state A to be %v, but got %v", expect, res)
	}

	// Add second node
	b := []*disco.Instance{
		&disco.Instance{
			ID:   "alpha",
			Name: "Instance Alpha",
			Host: "127.0.0.1",
			Port: 1001,
		},
		&disco.Instance{
			ID:   "beta",
			Name: "Instance Beta",
			Host: "127.0.0.1",
			Port: 1002,
		},
	}
	expect = []*disco.Event{
		&disco.Event{
			Op:       disco.Update,
			Instance: b[0], // TODO: Remove once diff tests when updates are needed
		},
		&disco.Event{
			Op:       disco.Add,
			Instance: b[1],
		},
	}

	res = diff.Apply(b)
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("expect state B to be %v, but got %v", expect, res)
	}

	// Remove first node
	c := []*disco.Instance{
		&disco.Instance{
			ID:   "beta",
			Name: "Instance Beta",
			Host: "127.0.0.1",
			Port: 1002,
		},
	}
	expect = []*disco.Event{
		&disco.Event{
			Op:       disco.Update,
			Instance: c[0], // TODO: Remove once diff tests when updates are needed
		},
		&disco.Event{
			Op: disco.Delete,
			Instance: &disco.Instance{
				ID:   "alpha",
				Name: "Instance Alpha",
				Host: "127.0.0.1",
				Port: 1001,
			},
		},
	}

	res = diff.Apply(c)
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("expect state C to be %v, but got %v", expect, res)
	}

	// Remove second node
	d := []*disco.Instance{}
	expect = []*disco.Event{
		&disco.Event{
			Op: disco.Delete,
			Instance: &disco.Instance{
				ID:   "beta",
				Name: "Instance Beta",
				Host: "127.0.0.1",
				Port: 1002,
			},
		},
	}

	res = diff.Apply(d)
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("expect state D to be %v, but got %v", expect, res)
	}
}
