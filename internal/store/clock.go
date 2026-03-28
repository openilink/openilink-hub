package store

import "time"

// Clock abstracts time.Now so store implementations can be tested with
// deterministic timestamps.
type Clock interface {
	Now() time.Time
}

// RealClock delegates to the standard library.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
