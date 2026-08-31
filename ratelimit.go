package main

import (
	"net"
	"sync"
	"time"
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter manages a token bucket per client IP
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	capacity   float64
	refillRate float64
}

// NewRateLimiter creates a RateLimiter with specified capacity and refill rate (tokens/sec)
func NewRateLimiter(capacity float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// Global rate limiter instance: 5 maximum tokens, 1 token/second refill rate
var globalRateLimiter = NewRateLimiter(5.0, 1.0)

// Allow checks if the given client IP is allowed to make a request
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	b, exists := rl.buckets[clientIP]
	if !exists {
		rl.buckets[clientIP] = &tokenBucket{
			tokens:     rl.capacity - 1.0,
			lastRefill: now,
		}
		return true
	}

	// Calculate elapsed time and add refilled tokens
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.refillRate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

// ExtractIP returns the host IP from a net.Addr
func ExtractIP(remoteAddr net.Addr) string {
	if remoteAddr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		return remoteAddr.String()
	}
	return host
}
