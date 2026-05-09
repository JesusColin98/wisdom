package cerebellum

import (
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is functioning normally and allowing calls.
	StateClosed CircuitState = iota
	// StateOpen means the circuit is tripped and blocking calls.
	StateOpen
	// StateHalfOpen means the circuit is testing if it can be closed again.
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern to provide fault tolerance.
type CircuitBreaker struct {
	failureThreshold    int
	resetTimeout        time.Duration
	consecutiveFailures int
	state               CircuitState
	lastFailureTime     time.Time
	mu                  sync.RWMutex
}

// NewCircuitBreaker creates a new CircuitBreaker with the specified threshold and timeout.
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: threshold,
		resetTimeout:     timeout,
		state:            StateClosed,
	}
}

// Allow returns true if the call should be permitted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		// In HalfOpen state, we only allow one trial call.
		// Since we don't have a way to track if a trial is in progress across multiple calls
		// without extra state, we transition back to Open if another Allow() is called
		// before success/failure is recorded, OR we can just return false here
		// if we want to be strict about "only one trial call".

		// If we are already in HalfOpen, it means someone is already trying.
		// To follow "only allow one trial call", we should return false for others.
		return false
	default:
		return false
	}
}

// RecordSuccess resets the consecutive failure count and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures = 0
	cb.state = StateClosed
}

// RecordFailure increments consecutive failures and trips the circuit if threshold is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen || cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}
