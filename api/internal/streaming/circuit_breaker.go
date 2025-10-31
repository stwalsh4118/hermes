package streaming

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed indicates the circuit is closed (normal operation)
	StateClosed CircuitState = iota
	// StateOpen indicates the circuit is open (blocking calls)
	StateOpen
	// StateHalfOpen indicates the circuit is testing if recovery is possible
	StateHalfOpen
)

const (
	stateUnknown = "unknown"
)

// String returns the string representation of CircuitState
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return stateUnknown
	}
}

// Circuit breaker errors
var (
	// ErrCircuitOpen indicates the circuit breaker is open and blocking calls
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	state            CircuitState
	failures         int
	lastFailureTime  time.Time
	mu               sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker with the given threshold and reset timeout
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
		failures:         0,
		lastFailureTime:  time.Time{},
	}
}

// Call executes the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if circuit should transition from Open to HalfOpen
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.mu.Unlock()
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	} else {
		cb.mu.Unlock()
	}

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailureLocked()
		return err
	}

	cb.recordSuccessLocked()
	return nil
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.recordSuccessLocked()
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.recordFailureLocked()
}

// recordSuccessLocked records a successful operation (must hold lock)
func (cb *CircuitBreaker) recordSuccessLocked() {
	cb.failures = 0
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
}

// recordFailureLocked records a failed operation (must hold lock)
func (cb *CircuitBreaker) recordFailureLocked() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Auto-transition from Open to HalfOpen if timeout elapsed
	if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.failures = 0
	}

	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.lastFailureTime = time.Time{}
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}

// CanAttempt returns true if the circuit breaker allows an attempt
func (cb *CircuitBreaker) CanAttempt() bool {
	state := cb.GetState()
	return state == StateClosed || state == StateHalfOpen
}
