package git

import (
	"context"
	"runtime"
	"sync"
	"testing"
)

func TestOperationCoordinatorPrioritizesUserAndSerializesPath(t *testing.T) {
	c := NewOperationCoordinator()
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var order []string

	go func() {
		_, _, _ = c.Run(context.Background(), "/repo", false, func(context.Context) error {
			mu.Lock()
			order = append(order, "running-low")
			mu.Unlock()
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	lowDone := make(chan bool, 1)
	go func() {
		_, ran, _ := c.Run(context.Background(), "/repo", false, func(context.Context) error {
			mu.Lock()
			order = append(order, "queued-low")
			mu.Unlock()
			return nil
		})
		lowDone <- ran
	}()
	for {
		c.mu.Lock()
		waiting := c.states["/repo"].lowWaiting
		c.mu.Unlock()
		if waiting > 0 {
			break
		}
		runtime.Gosched()
	}
	highDone := make(chan struct{})
	go func() {
		_, _, _ = c.Run(context.Background(), "/repo", true, func(context.Context) error {
			mu.Lock()
			order = append(order, "high")
			mu.Unlock()
			return nil
		})
		close(highDone)
	}()
	for {
		c.mu.Lock()
		waiting := c.states["/repo"].highWaiting
		c.mu.Unlock()
		if waiting > 0 {
			break
		}
		runtime.Gosched()
	}
	close(release)
	<-highDone
	if ran := <-lowDone; ran {
		t.Fatal("queued low-priority operation ran")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "running-low" || order[1] != "high" {
		t.Fatalf("order = %v", order)
	}
}

func TestOperationCoordinatorReleaseIsIdempotent(t *testing.T) {
	c := NewOperationCoordinator()
	_, ran, release, err := c.Acquire(context.Background(), "/repo", true)
	if err != nil || !ran {
		t.Fatalf("Acquire() ran=%v err=%v", ran, err)
	}
	release()
	release()
	if _, ran, nextRelease, err := c.Acquire(context.Background(), "/repo", true); err != nil || !ran {
		t.Fatalf("second Acquire() ran=%v err=%v", ran, err)
	} else {
		nextRelease()
	}
}
