package local

import (
	"sync"
	"time"

	pb "github.com/deixis/spine/schedule/adapter/local/localpb"
)

type watcher struct {
	mu sync.Mutex

	closing bool
	next    int64
	updatec chan struct{}
	stopc   chan struct{}

	processc chan<- *pb.Event
	storage  *storage
}

func newWatcher(storage *storage, processc chan<- *pb.Event) *watcher {
	return &watcher{
		updatec:  make(chan struct{}, 1),
		stopc:    make(chan struct{}),
		processc: processc,
		storage:  storage,
	}
}

func (w *watcher) Start() {
	go w.run()
}

func (w *watcher) Notify(t int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if t >= w.next || w.closing {
		return
	}
	w.next = t

	select {
	case w.updatec <- struct{}{}:
	default:
	}
}

// Close drains the bucket and stop
func (w *watcher) Close() error {
	w.mu.Lock()
	w.closing = true
	w.mu.Unlock()

	w.stopc <- struct{}{}
	close(w.stopc)
	return nil
}

func (w *watcher) run() {
	for {
		now := time.Now().UnixNano()
		events, next, err := w.storage.Load(now)
		switch err {
		case errDatabaseClosed:
			return
		}

		w.mu.Lock()
		w.next = next
		w.mu.Unlock()

		if len(events) == 0 {
			select {
			case <-time.Tick(time.Duration(next - now)):
				continue
			case <-w.updatec:
				continue
			case <-w.stopc:
				return
			}
		}
		for i := range events {
			w.processc <- events[i]
		}
	}
}
