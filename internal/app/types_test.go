package app

import (
	"testing"
	"time"
)

// TestNewEventThrottler tests the NewEventThrottler constructor
// which creates a new event throttler with the specified grace period
func TestNewEventThrottler(t *testing.T) {
	gracePeriod := 100 * time.Millisecond
	throttler := NewEventThrottler(gracePeriod)

	if throttler == nil {
		t.Fatal("NewEventThrottler returned nil")
	}

	if throttler.gracePeriod != gracePeriod {
		t.Errorf("Expected gracePeriod %v, got %v", gracePeriod, throttler.gracePeriod)
	}

	if throttler.C == nil {
		t.Error("Expected C channel to be initialized")
	}

	if throttler.timer != nil {
		t.Error("Expected timer to be nil on creation")
	}
}

// TestEventThrottlerNotify tests the EventThrottler.Notify method
// which triggers the throttling mechanism
func TestEventThrottlerNotify(t *testing.T) {
	gracePeriod := 50 * time.Millisecond
	throttler := NewEventThrottler(gracePeriod)

	// First notify should start the timer
	throttler.Notify()

	if throttler.timer == nil {
		t.Error("Expected timer to be set after Notify()")
	}

	// Second notify should be ignored (throttling)
	throttler.Notify()

	// Wait for grace period to expire
	time.Sleep(gracePeriod + 10*time.Millisecond)

	// Timer should be cleared after expiration
	if throttler.timer != nil {
		t.Error("Expected timer to be nil after grace period expired")
	}
}
