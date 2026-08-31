package main

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	threshold := 3
	cooldown := 50 * time.Millisecond
	cb := NewCircuitBreaker(threshold, cooldown)

	// 1. Initial State: CLOSED
	if cb.State() != StateClosed {
		t.Fatalf("expected initial state CLOSED, got %s", cb.State())
	}
	if !cb.AllowRequest() {
		t.Fatalf("expected AllowRequest to be true in CLOSED state")
	}

	// 2. Threshold Failures -> Transition to OPEN
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatalf("expected state CLOSED before reaching threshold, got %s", cb.State())
	}

	cb.RecordFailure() // 3rd failure reaches threshold
	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN after threshold failures, got %s", cb.State())
	}
	if cb.AllowRequest() {
		t.Fatalf("expected AllowRequest to be false in OPEN state")
	}

	// 3. Cooldown Period -> Transition to HALF-OPEN
	time.Sleep(cooldown + 10*time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state HALF-OPEN after cooldown, got %s", cb.State())
	}

	// First request after cooldown should be allowed (trial request)
	if !cb.AllowRequest() {
		t.Fatalf("expected trial request to be allowed in HALF-OPEN state")
	}
	// Second request while trial is active should be rejected
	if cb.AllowRequest() {
		t.Fatalf("expected additional request during active trial to be rejected")
	}

	// 4. Successful Trial -> CLOSED
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("expected state CLOSED after successful trial, got %s", cb.State())
	}
}

func TestCircuitBreaker_FailedRecovery(t *testing.T) {
	threshold := 2
	cooldown := 30 * time.Millisecond
	cb := NewCircuitBreaker(threshold, cooldown)

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN state, got %s", cb.State())
	}

	time.Sleep(cooldown + 10*time.Millisecond)

	// Allow trial request
	if !cb.AllowRequest() {
		t.Fatalf("expected trial request allowed")
	}

	// Failed trial -> Re-opens circuit
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN after failed trial, got %s", cb.State())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(5, 50*time.Millisecond)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				cb.AllowRequest()
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
				cb.State()
			}
		}(i)
	}

	wg.Wait()
}
