package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func startDummyBackend(t *testing.T, responseBody string) (*httptest.Server, string) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain")
		if len(body) > 0 {
			fmt.Fprintf(w, "%s (received body: %s)", responseBody, string(body))
		} else {
			fmt.Fprint(w, responseBody)
		}
	}))
	addr := ts.Listener.Addr().String()
	return ts, addr
}

func TestIntegration_ProxyRequestAndBody(t *testing.T) {
	backend, addr := startDummyBackend(t, "Backend 1 Hello")
	defer backend.Close()

	// Set up backend in global slice
	mu.Lock()
	backends = []*Backend{
		{
			Address: addr,
			Healthy: true,
			CB:      NewCircuitBreaker(3, 5*time.Second),
		},
	}
	mu.Unlock()

	// Dial dummy backend through forwardToBackend
	b := backends[0]
	rawReq := []byte("POST /test HTTP/1.1\r\nHost: localhost\r\nContent-Length: 11\r\n\r\nHello World")

	resp, conn, err := forwardToBackend(b, rawReq)
	if err != nil {
		t.Fatalf("forwardToBackend failed: %v", err)
	}
	defer conn.Close()

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !bytes.Contains(body, []byte("Backend 1 Hello")) || !bytes.Contains(body, []byte("Hello World")) {
		t.Fatalf("unexpected response body: %s", string(body))
	}
}

func TestIntegration_BackendFailover(t *testing.T) {
	backend1, addr1 := startDummyBackend(t, "Backend 1")
	defer backend1.Close()

	// Dead backend address
	addrDead := "127.0.0.1:59999"

	mu.Lock()
	backends = []*Backend{
		{
			Address: addrDead,
			Healthy: false,
			CB:      NewCircuitBreaker(3, 5*time.Second),
		},
		{
			Address: addr1,
			Healthy: true,
			CB:      NewCircuitBreaker(3, 5*time.Second),
		},
	}
	mu.Unlock()

	// getLeastConnectionsBackend should skip unhealthy/dead backend
	selected := getLeastConnectionsBackend()
	if selected == nil || selected.Address != addr1 {
		t.Fatalf("expected selected backend to be %s, got %v", addr1, selected)
	}
}
