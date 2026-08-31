package main

import (
	"fmt"
	"net/http"
	"sync"
)

// Metrics tracks request counts and errors for the proxy
type Metrics struct {
	mu            sync.Mutex
	totalRequests int64
	totalErrors   int64
	backendReqs   map[string]int64
}

// NewMetrics creates and initializes a Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		backendReqs: make(map[string]int64),
	}
}

// Global metrics instance
var globalMetrics = NewMetrics()

// RecordRequest records a processed request and updates metrics
func (m *Metrics) RecordRequest(backend string, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	if isError {
		m.totalErrors++
	}
	if backend != "" {
		m.backendReqs[backend]++
	}
}

// MetricsHandler serves metrics in Prometheus-compatible plain-text format
func (m *Metrics) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintln(w, "# HELP proxy_requests_total Total number of HTTP requests processed by the proxy.")
	fmt.Fprintln(w, "# TYPE proxy_requests_total counter")
	fmt.Fprintf(w, "proxy_requests_total %d\n\n", m.totalRequests)

	fmt.Fprintln(w, "# HELP proxy_errors_total Total number of HTTP request errors encountered by the proxy.")
	fmt.Fprintln(w, "# TYPE proxy_errors_total counter")
	fmt.Fprintf(w, "proxy_errors_total %d\n\n", m.totalErrors)

	fmt.Fprintln(w, "# HELP proxy_backend_requests_total Total number of requests sent to each backend.")
	fmt.Fprintln(w, "# TYPE proxy_backend_requests_total counter")
	for backend, count := range m.backendReqs {
		fmt.Fprintf(w, "proxy_backend_requests_total{backend=\"%s\"} %d\n", backend, count)
	}
}
