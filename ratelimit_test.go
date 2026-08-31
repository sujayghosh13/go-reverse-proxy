package main

import (
	"net"
	"testing"
	"time"
)

func TestRateLimiter_TokenBucketBehavior(t *testing.T) {
	// Capacity 2, refill rate 1 token/sec
	rl := NewRateLimiter(2.0, 1.0)
	clientIP := "192.168.1.100"

	// Request 1: allowed (uses 1 token, 1 remaining)
	if !rl.Allow(clientIP) {
		t.Fatalf("expected request 1 to be allowed")
	}

	// Request 2: allowed (uses 1 token, 0 remaining)
	if !rl.Allow(clientIP) {
		t.Fatalf("expected request 2 to be allowed")
	}

	// Request 3: denied (0 tokens remaining)
	if rl.Allow(clientIP) {
		t.Fatalf("expected request 3 to be denied")
	}

	// Wait 1.1s for 1 token refill
	time.Sleep(1100 * time.Millisecond)

	// Request 4: allowed after refill
	if !rl.Allow(clientIP) {
		t.Fatalf("expected request 4 to be allowed after token refill")
	}
}

func TestExtractIP(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:54321")
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	ip := ExtractIP(addr)
	if ip != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %s", ip)
	}

	if ExtractIP(nil) != "" {
		t.Fatalf("expected empty string for nil addr")
	}
}
