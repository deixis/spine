package local

import (
	pb "github.com/deixis/spine/schedule/adapter/local/localpb"
)

// processor spawns a pool of goroutines to process events in parallel
type processor struct {
	n       int
	bucketc chan *pb.Event
	process func(e *pb.Event)
}

func newProcessor(n int, fn func(e *pb.Event)) *processor {
	return &processor{
		n:       n,
		bucketc: make(chan *pb.Event),
		process: fn,
	}
}

func (p *processor) Start() {
	for i := 0; i < p.n; i++ {
		go func() {
			var stop bool
			for !stop {
				stop = p.run()
			}
		}()
	}
}

func (p *processor) Exec() chan<- *pb.Event {
	return p.bucketc
}

// Close drains the bucket and stop
func (p *processor) Close() error {
	for i := 0; i < p.n; i++ {
		p.bucketc <- nil // Stop all goroutines
	}
	close(p.bucketc)
	return nil
}

func (p *processor) run() (stop bool) {
	defer func() {
		recover()
	}()

	for {
		select {
		case e := <-p.bucketc:
			if e == nil {
				stop = true
				return stop
			}
			p.process(e)
		}
	}
}
