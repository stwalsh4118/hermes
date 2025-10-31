package streaming

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    CircuitState
		expected string
	}{
		{"Closed", StateClosed, "closed"},
		{"Open", StateOpen, "open"},
		{"Half Open", StateHalfOpen, "half_open"},
		{"Unknown", CircuitState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("CircuitState.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	threshold := 3
	resetTimeout := 10 * time.Second

	cb := NewCircuitBreaker(threshold, resetTimeout)

	if cb.failureThreshold != threshold {
		t.Errorf("failureThreshold = %v, want %v", cb.failureThreshold, threshold)
	}
	if cb.resetTimeout != resetTimeout {
		t.Errorf("resetTimeout = %v, want %v", cb.resetTimeout, resetTimeout)
	}
	if cb.state != StateClosed {
		t.Errorf("initial state = %v, want %v", cb.state, StateClosed)
	}
	if cb.failures != 0 {
		t.Errorf("initial failures = %v, want 0", cb.failures)
	}
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Call() error = %v, want nil", err)
	}
	if !called {
		t.Error("Function should have been called")
	}
	if cb.GetState() != StateClosed {
		t.Errorf("State = %v, want %v", cb.GetState(), StateClosed)
	}
	if cb.GetFailures() != 0 {
		t.Errorf("Failures = %v, want 0", cb.GetFailures())
	}
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	testErr := errors.New("test error")
	err := cb.Call(func() error {
		return testErr
	})

	if !errors.Is(err, testErr) {
		t.Errorf("Call() error = %v, want %v", err, testErr)
	}
	if cb.GetFailures() != 1 {
		t.Errorf("Failures = %v, want 1", cb.GetFailures())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("State = %v, want %v (should still be closed after 1 failure)", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	testErr := errors.New("test error")

	// First 2 failures - should stay closed
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error { return testErr })
		if cb.GetState() != StateClosed {
			t.Errorf("After %d failures, state = %v, want %v", i+1, cb.GetState(), StateClosed)
		}
	}

	// 3rd failure - should open
	_ = cb.Call(func() error { return testErr })
	if cb.GetState() != StateOpen {
		t.Errorf("After 3 failures, state = %v, want %v", cb.GetState(), StateOpen)
	}
	if cb.GetFailures() != 3 {
		t.Errorf("Failures = %v, want 3", cb.GetFailures())
	}
}

func TestCircuitBreaker_BlocksWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Second)

	// Trigger the circuit to open
	_ = cb.Call(func() error { return errors.New("error") })

	// Try to call again - should be blocked
	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Call() error = %v, want %v", err, ErrCircuitOpen)
	}
	if called {
		t.Error("Function should not have been called when circuit is open")
	}
}

func TestCircuitBreaker_TransitionToHalfOpen(t *testing.T) {
	resetTimeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(1, resetTimeout)

	// Open the circuit
	_ = cb.Call(func() error { return errors.New("error") })
	if cb.GetState() != StateOpen {
		t.Fatalf("Circuit should be open")
	}

	// Wait for reset timeout
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// GetState should transition to HalfOpen
	state := cb.GetState()
	if state != StateHalfOpen {
		t.Errorf("After reset timeout, state = %v, want %v", state, StateHalfOpen)
	}
}

func TestCircuitBreaker_HalfOpenAllowsAttempt(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(1, resetTimeout)

	// Open the circuit
	_ = cb.Call(func() error { return errors.New("error") })

	// Wait for reset timeout
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// Now call should be allowed (HalfOpen)
	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Call() error = %v, want nil", err)
	}
	if !called {
		t.Error("Function should have been called in HalfOpen state")
	}
	if cb.GetState() != StateClosed {
		t.Errorf("After successful call in HalfOpen, state = %v, want %v", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(1, resetTimeout)

	// Open the circuit
	_ = cb.Call(func() error { return errors.New("error") })

	// Wait for reset timeout
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// Call with failure - should reopen
	testErr := errors.New("second error")
	err := cb.Call(func() error { return testErr })

	if !errors.Is(err, testErr) {
		t.Errorf("Call() error = %v, want %v", err, testErr)
	}
	if cb.GetState() != StateOpen {
		t.Errorf("After failure in HalfOpen, state = %v, want %v", cb.GetState(), StateOpen)
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetFailures() != 2 {
		t.Fatalf("Failures = %v, want 2", cb.GetFailures())
	}

	// Record success - should reset failures
	cb.RecordSuccess()

	if cb.GetFailures() != 0 {
		t.Errorf("After RecordSuccess, failures = %v, want 0", cb.GetFailures())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("After RecordSuccess, state = %v, want %v", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)

	cb.RecordFailure()
	if cb.GetFailures() != 1 {
		t.Errorf("After 1 RecordFailure, failures = %v, want 1", cb.GetFailures())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("After 1 failure, state = %v, want %v", cb.GetState(), StateClosed)
	}

	cb.RecordFailure()
	if cb.GetFailures() != 2 {
		t.Errorf("After 2 RecordFailure, failures = %v, want 2", cb.GetFailures())
	}
	if cb.GetState() != StateOpen {
		t.Errorf("After threshold failures, state = %v, want %v", cb.GetState(), StateOpen)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Second)

	// Open the circuit
	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Fatalf("Circuit should be open")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("After Reset, state = %v, want %v", cb.GetState(), StateClosed)
	}
	if cb.GetFailures() != 0 {
		t.Errorf("After Reset, failures = %v, want 0", cb.GetFailures())
	}
}

func TestCircuitBreaker_IsOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Second)

	if cb.IsOpen() {
		t.Error("Circuit should not be open initially")
	}

	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Error("Circuit should be open after threshold failures")
	}

	cb.Reset()
	if cb.IsOpen() {
		t.Error("Circuit should not be open after reset")
	}
}

func TestCircuitBreaker_CanAttempt(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	// Initially closed - can attempt
	if !cb.CanAttempt() {
		t.Error("CanAttempt should be true when closed")
	}

	// Open the circuit
	cb.RecordFailure()
	if cb.CanAttempt() {
		t.Error("CanAttempt should be false when open")
	}

	// Wait for reset timeout - should transition to HalfOpen
	time.Sleep(60 * time.Millisecond)
	if !cb.CanAttempt() {
		t.Error("CanAttempt should be true when half-open")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(10, 100*time.Millisecond)

	// Run multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = cb.Call(func() error {
					if j%2 == 0 {
						return errors.New("error")
					}
					return nil
				})
			}
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Circuit breaker should still be in a valid state
	state := cb.GetState()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("Invalid state after concurrent access: %v", state)
	}
}

func TestCircuitBreaker_SuccessResetsInHalfOpen(t *testing.T) {
	resetTimeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(1, resetTimeout)

	// Open circuit
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(resetTimeout + 10*time.Millisecond)

	// Verify we're in half-open
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("Expected half-open state, got %v", cb.GetState())
	}

	// Success should close the circuit
	cb.RecordSuccess()

	if cb.GetState() != StateClosed {
		t.Errorf("After success in half-open, state = %v, want %v", cb.GetState(), StateClosed)
	}
}
