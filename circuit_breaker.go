package main

import (
	"sync"
	"time"
)

type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF-OPEN"
)

type CircuitBreaker struct {
	mu                  sync.Mutex
	state               CircuitState
	consecutiveFailures int
	failureThreshold    int
	cooldownPeriod      time.Duration
	lastStateChange     time.Time
	halfOpenTrialActive bool
	totalTripped        int64
}

func NewCircuitBreaker(failureThreshold int, cooldownPeriod time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if cooldownPeriod <= 0 {
		cooldownPeriod = 5 * time.Second
	}
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		cooldownPeriod:   cooldownPeriod,
		lastStateChange:  time.Now(),
	}
}

// AllowRequest checks if a request can be routed to this backend.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastStateChange) >= cb.cooldownPeriod {
			cb.state = StateHalfOpen
			cb.lastStateChange = now
			cb.halfOpenTrialActive = true
			return true
		}
		return false
	case StateHalfOpen:
		if !cb.halfOpenTrialActive {
			cb.halfOpenTrialActive = true
			return true
		}
		return false
	default:
		return true
	}
}

// RecordSuccess is called when a request to the backend succeeds.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures = 0
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.lastStateChange = time.Now()
		cb.halfOpenTrialActive = false
	}
}

// RecordFailure is called when a request to the backend fails.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	if cb.state == StateClosed && cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		cb.totalTripped++
	} else if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		cb.halfOpenTrialActive = false
		cb.totalTripped++
	}
}

// State returns current state and checks for cooldown expiration.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.lastStateChange) >= cb.cooldownPeriod {
		return StateHalfOpen
	}
	return cb.state
}

// TotalTripped returns total times the circuit breaker tripped to open.
func (cb *CircuitBreaker) TotalTripped() int64 {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.totalTripped
}
