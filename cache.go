package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CacheItem struct {
	ResponseData []byte
	StatusCode   int
	CachedAt     time.Time
}

type ResponseCache struct {
	mu      sync.RWMutex
	items   map[string]*CacheItem
	ttl     time.Duration
	enabled bool
	hits    int64
	misses  int64
}

func NewResponseCache(enabled bool, ttl time.Duration) *ResponseCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &ResponseCache{
		items:   make(map[string]*CacheItem),
		ttl:     ttl,
		enabled: enabled,
	}
}

var globalCache = NewResponseCache(false, 30*time.Second)

func (c *ResponseCache) Get(key string) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		c.misses++
		return nil, false
	}

	if time.Since(item.CachedAt) > c.ttl {
		delete(c.items, key)
		c.misses++
		return nil, false
	}

	c.hits++
	return item.ResponseData, true
}

func (c *ResponseCache) Put(key string, data []byte, statusCode int) {
	if !c.enabled || statusCode != 200 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		ResponseData: data,
		StatusCode:   statusCode,
		CachedAt:     time.Now(),
	}
}

func (c *ResponseCache) Hits() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits
}

func (c *ResponseCache) Misses() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.misses
}

// ExtractCacheKey returns method + URI for GET requests, or empty string if uncacheable
func ExtractCacheKey(rawRequest []byte) (string, string) {
	lines := strings.Split(string(rawRequest), "\r\n")
	if len(lines) == 0 {
		return "", ""
	}
	parts := strings.Split(lines[0], " ")
	if len(parts) < 2 {
		return "", ""
	}
	method := parts[0]
	uri := parts[1]

	if method == "GET" {
		return fmt.Sprintf("GET %s", uri), method
	}
	return "", method
}

// BuildResponseBytes formats an http.Response into raw HTTP bytes for caching
func BuildResponseBytes(resp *http.Response, body []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "HTTP/1.1 %s\r\n", resp.Status)
	for k, vv := range resp.Header {
		for _, v := range vv {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(body))
	buf.Write(body)
	return buf.Bytes()
}
