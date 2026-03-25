package mockserver

import (
	"testing"
	"time"
)

func TestFakeClock_Now(t *testing.T) {
	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := NewFakeClock(epoch)
	if got := fc.Now(); !got.Equal(epoch) {
		t.Fatalf("Now() = %v, want %v", got, epoch)
	}
}

func TestFakeClock_AdvanceFiresTimer(t *testing.T) {
	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := NewFakeClock(epoch)

	timer := fc.NewTimer(5 * time.Second)

	// Timer should not have fired yet.
	select {
	case <-timer.C:
		t.Fatal("timer fired before advance")
	default:
	}

	fc.Advance(5 * time.Second)

	select {
	case got := <-timer.C:
		want := epoch.Add(5 * time.Second)
		if !got.Equal(want) {
			t.Fatalf("timer fired with %v, want %v", got, want)
		}
	default:
		t.Fatal("timer did not fire after advance")
	}
}

func TestFakeClock_StopPreventsFireing(t *testing.T) {
	fc := NewFakeClock(time.Now())
	timer := fc.NewTimer(10 * time.Second)

	if !timer.Stop() {
		t.Fatal("Stop() returned false on unfired timer")
	}

	fc.Advance(10 * time.Second)

	select {
	case <-timer.C:
		t.Fatal("stopped timer should not fire")
	default:
	}

	// Second stop should return false.
	if timer.Stop() {
		t.Fatal("second Stop() should return false")
	}
}

func TestFakeClock_After(t *testing.T) {
	epoch := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	fc := NewFakeClock(epoch)

	ch := fc.After(3 * time.Second)
	fc.Advance(3 * time.Second)

	select {
	case <-ch:
		// OK
	default:
		t.Fatal("After channel did not receive")
	}
}

func TestFakeClock_MultipleTimers(t *testing.T) {
	fc := NewFakeClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	t1 := fc.NewTimer(2 * time.Second)
	t2 := fc.NewTimer(5 * time.Second)
	t3 := fc.NewTimer(5 * time.Second)

	fc.Advance(3 * time.Second)

	// t1 should fire
	select {
	case <-t1.C:
	default:
		t.Fatal("t1 should have fired")
	}

	// t2, t3 should not fire yet
	select {
	case <-t2.C:
		t.Fatal("t2 should not have fired yet")
	default:
	}
	select {
	case <-t3.C:
		t.Fatal("t3 should not have fired yet")
	default:
	}

	fc.Advance(2 * time.Second)

	// Now t2 and t3 should fire
	select {
	case <-t2.C:
	default:
		t.Fatal("t2 should have fired")
	}
	select {
	case <-t3.C:
	default:
		t.Fatal("t3 should have fired")
	}
}

func TestFakeClock_Reset(t *testing.T) {
	fc := NewFakeClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	timer := fc.NewTimer(2 * time.Second)

	// Reset to 10 seconds from now.
	timer.Reset(10 * time.Second)

	fc.Advance(5 * time.Second)
	select {
	case <-timer.C:
		t.Fatal("timer should not have fired after reset")
	default:
	}

	fc.Advance(5 * time.Second)
	select {
	case <-timer.C:
		// OK
	default:
		t.Fatal("timer should have fired after full duration")
	}
}

func TestFakeClock_Set(t *testing.T) {
	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := NewFakeClock(epoch)
	timer := fc.NewTimer(5 * time.Second)

	fc.Set(epoch.Add(5 * time.Second))

	select {
	case <-timer.C:
	default:
		t.Fatal("Set should have advanced and fired timer")
	}

	// Setting to the past is a no-op.
	before := fc.Now()
	fc.Set(epoch)
	if !fc.Now().Equal(before) {
		t.Fatal("Set to past should be a no-op")
	}
}

// Verify FakeClock satisfies the Clock interface.
var _ Clock = (*FakeClock)(nil)
