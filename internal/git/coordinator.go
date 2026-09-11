package git

import (
	"context"
	"sync"
)

// OperationCoordinator serializes discovery fetches and user-requested remote
// operations per repository. A waiting high-priority operation causes queued
// low-priority discovery work to be discarded.
type OperationCoordinator struct {
	mu     sync.Mutex
	states map[string]*operationState
}

type operationState struct {
	running        bool
	highWaiting    int
	lowWaiting     int
	highGeneration uint64
	revision       uint64
	done           chan struct{}
}

func NewOperationCoordinator() *OperationCoordinator {
	return &OperationCoordinator{states: make(map[string]*operationState)}
}

// Acquire reserves exclusive access to key. The caller must invoke the
// returned release function exactly once after its operation completes.
// It returns false when a low-priority operation was superseded while queued
// behind an existing operation.
func (c *OperationCoordinator) Acquire(ctx context.Context, key string, high bool) (revision uint64, ran bool, release func(), err error) {
	queuedHigh := false
	queuedLow := false
	lowGeneration := uint64(0)
	for {
		c.mu.Lock()
		s := c.states[key]
		if s == nil {
			s = &operationState{}
			c.states[key] = s
		}
		if !s.running {
			if !high && (s.highWaiting > 0 || (queuedLow && s.highGeneration != lowGeneration)) {
				c.mu.Unlock()
				return s.revision, false, nil, nil
			}
			if queuedHigh {
				s.highWaiting--
			}
			s.running = true
			s.revision++
			s.done = make(chan struct{})
			revision = s.revision
			c.mu.Unlock()
			return revision, true, func() {
				c.mu.Lock()
				s.running = false
				close(s.done)
				c.mu.Unlock()
			}, nil
		}
		wait := s.done
		if high && !queuedHigh {
			s.highWaiting++
			s.highGeneration++
			queuedHigh = true
		} else if !high {
			if !queuedLow {
				lowGeneration = s.highGeneration
				queuedLow = true
			}
			s.lowWaiting++
		}
		c.mu.Unlock()
		select {
		case <-wait:
			if !high {
				c.mu.Lock()
				c.states[key].lowWaiting--
				c.mu.Unlock()
			}
		case <-ctx.Done():
			if queuedHigh {
				c.mu.Lock()
				c.states[key].highWaiting--
				c.mu.Unlock()
			}
			if !high {
				c.mu.Lock()
				c.states[key].lowWaiting--
				c.mu.Unlock()
			}
			return c.Revision(key), false, nil, ctx.Err()
		}
		if !high {
			c.mu.Lock()
			superseded := c.states[key].highWaiting > 0
			revision = c.states[key].revision
			c.mu.Unlock()
			if superseded {
				return revision, false, nil, nil
			}
		}
	}
}

// Run executes fn exclusively for key.
func (c *OperationCoordinator) Run(ctx context.Context, key string, high bool, fn func(context.Context) error) (revision uint64, ran bool, err error) {
	revision, ran, release, err := c.Acquire(ctx, key, high)
	if err != nil || !ran {
		return revision, ran, err
	}
	defer release()
	return revision, true, fn(ctx)
}

func (c *OperationCoordinator) Revision(path string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := c.states[path]; s != nil {
		return s.revision
	}
	return 0
}

func (c *OperationCoordinator) IsCurrent(path string, revision uint64) bool {
	return c.Revision(path) == revision
}
