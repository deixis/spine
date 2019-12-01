package bg_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/deixis/spine/bg"
)

// TestJobRegistration tests whether a running job can be registered once and
// only once
func TestJobRegistration(t *testing.T) {
	reg := bg.NewReg("TestJobRegistration", context.Background())
	job := NewDummyJob(time.Microsecond)

	if err := reg.Dispatch(job); err != nil {
		t.Error("expect to be able to start job after its first registration", err)
	}

	if err := reg.Dispatch(job); err != bg.ErrDup {
		t.Error("expect registration to fail when the same job is registered twice")
	}
}

// TestJobDeregistration tests whether a job can be registered once again
// after its completion
func TestJobDeregistration(t *testing.T) {
	reg := bg.NewReg("TestJobDeregistration", context.Background())
	job := NewDummyJob(time.Microsecond)
	job.exec = time.Microsecond * 5

	if err := reg.Dispatch(job); err != nil {
		t.Error("expect to be able to start job after its first registration", err)
	}

	<-job.done                      // wait for job to stop
	time.Sleep(5 * time.Nanosecond) // The deregistration goroutine, might not be executed otherwise
	job.reset()

	if err := reg.Dispatch(job); err != nil {
		t.Errorf("expect to be able to start job again once deregistered (%s)", err)
	}
}

// TestJobRegistrationDrainMode tests whether jobs are properly rejected
// when the registry is in drain mode
func TestJobRegistrationDrainMode(t *testing.T) {
	reg := bg.NewReg("TestJobRegistrationDrainMode", context.Background())

	// Set registry to drain mode
	reg.Drain()

	// Attempt to register job
	if err := reg.Dispatch(NewDummyJob(0)); err != bg.ErrDrain {
		t.Errorf("expect to reject job when registry is in drain mode (%s)", err)
	}
}

// TestJobDraining tests whether the registry drains properly the in-flight jobs
func TestJobDraining(t *testing.T) {
	reg := bg.NewReg("TestJobDraining", context.Background())

	jobs := []*DummyJob{
		NewDummyJob(0),
		NewDummyJob(0),
		NewDummyJob(time.Microsecond),
		NewDummyJob(time.Microsecond * 2),
		NewDummyJob(time.Microsecond * 4),
		NewDummyJob(time.Microsecond * 16),
		NewDummyJob(time.Microsecond * 256),
	}

	// Register all jobs
	for _, j := range jobs {
		if err := reg.Dispatch(j); err != nil {
			t.Error("expect to be able to start job", err)
		}
	}

	// Start draining
	reg.Drain()

	// Ensure that all jobs have drained
	for i, j := range jobs {
		if j.running {
			t.Errorf("job <%d : %p> - is still running (%d)", i, j, j.drain)
		}
	}
}

type DummyJob struct {
	mu      sync.Mutex
	started bool          // has been started
	stopped bool          // has been stopped
	running bool          // is currently running
	exec    time.Duration // simulate time to execute the job (0 = infinite)
	drain   time.Duration // simulate time to drain the job
	stop    chan struct{} // ask job to stop
	done    chan struct{} // signal that a job has stopped
}

func NewDummyJob(d time.Duration) *DummyJob {
	return &DummyJob{
		drain: d,
		stop:  make(chan struct{}),
		done:  make(chan struct{}, 1),
	}
}

func (j *DummyJob) Start() {
	j.mu.Lock()
	j.started = true
	j.running = true

	if j.exec == 0 {
		j.mu.Unlock()
		<-j.stop
		time.Sleep(j.drain)
	} else {
		j.mu.Unlock()
		time.Sleep(j.exec)
	}

	j.mu.Lock()
	j.running = false
	j.done <- struct{}{}
	j.mu.Unlock()
}

func (j *DummyJob) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.started {
		// There is a slim chance to have stop called before start
		// It is not a big deal, since it is not guarantee.
		// However, we want tests to fail when it happens
		panic("Stop called before Start. (It is not a big issue)")
	}

	j.stopped = true
	if j.exec != 0 {
		j.stop <- struct{}{}
	}
	j.running = false
}

func (j *DummyJob) reset() {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.started = false
	j.stopped = false
	j.running = false
	j.done = make(chan struct{}, 1)
}
