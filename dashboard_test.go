package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rr := httptest.NewRecorder()

	DashboardHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<title>Go Reverse Proxy Dashboard</title>") {
		t.Fatalf("expected dashboard HTML content, got:\n%s", rr.Body.String())
	}
}

func TestStatusAPIHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/status", nil)
	rr := httptest.NewRecorder()

	StatusAPIHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var status StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
}
