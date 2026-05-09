package cerebellum

import (
	"testing"
	"time"
)

func TestCircuitBreaker_BasicFlow(t *testing.T) {
	threshold := 3
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(threshold, timeout)

	// Initially closed
	if !cb.Allow() {
		t.Fatal("expected circuit to be closed initially")
	}

	// Record failures up to threshold-1
	for i := 0; i < threshold-1; i++ {
		cb.RecordFailure()
		if !cb.Allow() {
			t.Fatalf("expected circuit to still be closed after %d failures", i+1)
		}
	}

	// Trip the circuit
	cb.RecordFailure()
	if cb.Allow() {
		t.Fatal("expected circuit to be open after threshold failures")
	}

	// Should still be open
	if cb.Allow() {
		t.Fatal("expected circuit to stay open")
	}

	// Wait for timeout
	time.Sleep(timeout + 10*time.Millisecond)

	// Should allow one call in HalfOpen
	if !cb.Allow() {
		t.Fatal("expected circuit to allow a trial call in HalfOpen state")
	}

	// Subsequent calls in HalfOpen should be blocked
	if cb.Allow() {
		t.Fatal("expected circuit to block subsequent calls in HalfOpen state")
	}

	// Record success should close the circuit
	cb.RecordSuccess()
	if !cb.Allow() {
		t.Fatal("expected circuit to be closed after recording success in HalfOpen")
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	threshold := 2
	timeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(threshold, timeout)

	cb.RecordFailure()
	cb.RecordFailure() // Trip

	if cb.Allow() {
		t.Fatal("expected circuit to be open")
	}

	time.Sleep(timeout + 10*time.Millisecond)

	if !cb.Allow() {
		t.Fatal("expected circuit to allow trial call")
	}

	// Failure in HalfOpen should immediately trip it again
	cb.RecordFailure()
	if cb.Allow() {
		t.Fatal("expected circuit to be open after failure in HalfOpen")
	}
}

func TestCircuitBreaker_ConsecutiveFailuresReset(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // Should reset counter

	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.Allow() {
		t.Fatal("expected circuit to be closed because counter was reset")
	}

	cb.RecordFailure() // This should trip it (3rd since reset)
	if cb.Allow() {
		t.Fatal("expected circuit to be open")
	}
}
