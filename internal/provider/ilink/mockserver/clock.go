package mockserver

import (
	"sync"
	"time"
)

// Clock abstracts time operations so tests can control time progression.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	NewTimer(d time.Duration) *ClockTimer
}

// ClockTimer wraps a timer channel with stop/reset capabilities.
type ClockTimer struct {
	C     <-chan time.Time
	stop  func() bool
	reset func(time.Duration)
}

// Stop prevents the timer from firing. It returns true if the call stops the
// timer, false if the timer has already been stopped or fired.
func (t *ClockTimer) Stop() bool { return t.stop() }

// Reset changes the timer to expire after duration d.
func (t *ClockTimer) Reset(d time.Duration) { t.reset(d) }

// realClock delegates to the standard library.
type realClock struct{}

func (realClock) Now() time.Time                        { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (realClock) NewTimer(d time.Duration) *ClockTimer {
	t := time.NewTimer(d)
	return &ClockTimer{C: t.C, stop: t.Stop, reset: func(d time.Duration) { t.Reset(d) }}
}

// FakeClock is a deterministic clock for testing. Time only advances when
// Advance or Set is called.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*fakeWaiter
}

type fakeWaiter struct {
	deadline time.Time
	ch       chan time.Time
	fired    bool
}

// NewFakeClock returns a FakeClock whose current time is now.
func NewFakeClock(now time.Time) *FakeClock { return &FakeClock{now: now} }

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	return c.NewTimer(d).C
}

func (c *FakeClock) NewTimer(d time.Duration) *ClockTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	w := &fakeWaiter{deadline: c.now.Add(d), ch: make(chan time.Time, 1)}
	c.waiters = append(c.waiters, w)
	ct := &ClockTimer{
		C: w.ch,
		stop: func() bool {
			c.mu.Lock()
			defer c.mu.Unlock()
			if w.fired {
				return false
			}
			w.fired = true
			return true
		},
		reset: func(nd time.Duration) {
			c.mu.Lock()
			defer c.mu.Unlock()
			w.deadline = c.now.Add(nd)
			w.fired = false
		},
	}
	return ct
}

// Advance moves the clock forward by d and fires any timers whose deadlines
// have been reached.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	var remaining []*fakeWaiter
	for _, w := range c.waiters {
		if !w.fired && !w.deadline.After(now) {
			w.fired = true
			select {
			case w.ch <- now:
			default:
			}
		}
		if !w.fired {
			remaining = append(remaining, w)
		}
	}
	c.waiters = remaining
	c.mu.Unlock()
}

// Set moves the clock to time t. If t is in the past relative to the current
// fake time, Set is a no-op.
func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	d := t.Sub(c.now)
	c.mu.Unlock()
	if d > 0 {
		c.Advance(d)
	}
}
