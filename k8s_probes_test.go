package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	HealthzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "OK\n" {
		t.Fatalf("expected body 'OK\\n', got %s", rr.Body.String())
	}
}

func TestReadyzHandler_HealthyBackends(t *testing.T) {
	mu.Lock()
	backends = []*Backend{
		{
			Address: "localhost:9001",
			Healthy: true,
			CB:      NewCircuitBreaker(3, 5*time.Second),
		},
	}
	mu.Unlock()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()

	ReadyzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "READY\n" {
		t.Fatalf("expected body 'READY\\n', got %s", rr.Body.String())
	}
}

func TestReadyzHandler_NoHealthyBackends(t *testing.T) {
	mu.Lock()
	backends = []*Backend{
		{
			Address: "localhost:9001",
			Healthy: false,
			CB:      NewCircuitBreaker(3, 5*time.Second),
		},
	}
	mu.Unlock()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()

	ReadyzHandler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when no healthy backends, got %d", rr.Code)
	}
}
