package main

import (
	"testing"
	"time"
)

func TestResponseCache_HitsAndMisses(t *testing.T) {
	cache := NewResponseCache(true, 50*time.Millisecond)
	key := "GET /test"
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK")

	// 1. Initial Miss
	if _, ok := cache.Get(key); ok {
		t.Fatalf("expected initial cache miss")
	}

	// 2. Put & Hit
	cache.Put(key, data, 200)
	cached, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected cache hit after Put")
	}
	if string(cached) != string(data) {
		t.Fatalf("expected cached data %s, got %s", data, cached)
	}

	// Verify hit/miss counts
	if cache.Hits() != 1 || cache.Misses() != 1 {
		t.Fatalf("expected 1 hit, 1 miss, got %d hits, %d misses", cache.Hits(), cache.Misses())
	}

	// 3. TTL Expiration
	time.Sleep(60 * time.Millisecond)
	if _, ok := cache.Get(key); ok {
		t.Fatalf("expected cache miss after TTL expiration")
	}
}

func TestResponseCache_OnlyCachesGETand200(t *testing.T) {
	cache := NewResponseCache(true, 1*time.Second)

	// Non-200 OK response should not be cached
	cache.Put("GET /error", []byte("HTTP/1.1 500 Error"), 500)
	if _, ok := cache.Get("GET /error"); ok {
		t.Fatalf("expected non-200 response to not be cached")
	}

	// Uncacheable method check
	key, method := ExtractCacheKey([]byte("POST /data HTTP/1.1\r\n\r\n"))
	if key != "" || method != "POST" {
		t.Fatalf("expected POST method to return empty cache key")
	}
}
