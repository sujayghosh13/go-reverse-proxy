# Go Reverse Proxy & Load Balancer — Built From Scratch

![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Raw_TCP_Sockets-FF6C37?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![CI/CD](https://img.shields.io/badge/CI%2FCD-GitHub_Actions-blue?style=for-the-badge&logo=githubactions)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Probes_%26_Manifests-326CE5?style=for-the-badge&logo=kubernetes)

> **Repository Description:** A high-performance, resilient HTTP/HTTPS Reverse Proxy & Load Balancer built in Go from raw TCP sockets (`net.Listen`, `net.Dial`, `crypto/tls`). Features least-connections load balancing, per-backend circuit breaking, token-bucket rate limiting, response caching, structured logging (`log/slog`), Prometheus metrics, and a real-time web dashboard.

A production-oriented reverse proxy and load balancer implemented in Go directly from raw TCP sockets (`net.Listen`, `net.Dial`, `crypto/tls`). 

The core proxy forwarding engine **does not use Go's standard `net/http` reverse-proxy package (`httputil.ReverseProxy`)** or third-party proxy libraries. It was built from first principles to gain a deep systems-level understanding of the mechanics powering enterprise edge proxies like Nginx and HAProxy — including TCP socket networking, HTTP protocol parsing, concurrency synchronization, load balancing algorithms, health checks, circuit breaking, connection pooling, rate limiting, TLS termination, structured logging, observability, and graceful application lifecycle management.

---

## 🎯 Why This Project Exists

Rather than wrapping pre-built HTTP proxy utilities, this project implements the proxy pipeline directly on top of Go's networking primitives. 

This project demonstrates practical mastery of:
- **HTTP Protocol over TCP:** Parsing raw request lines, headers, chunking, and streaming body payloads (`Content-Length`) over socket streams.
- **Dual-Role Proxy Networking:** Functioning simultaneously as an HTTPS server to downstream clients and a TCP/HTTP client to upstream backends.
- **Concurrent Connection Handling:** Managing thousands of simultaneous goroutine connections safely.
- **State Synchronization & Thread Safety:** Preventing data races using `sync.Mutex`, `sync.RWMutex`, and `sync.WaitGroup`.
- **Intelligent Load Balancing:** Implementing Round-Robin and Least-Connections distribution.
- **Resilience & Fault Tolerance:** Active background health probes and per-backend Circuit Breakers.
- **Connection Pooling:** Reducing TCP handshake overhead by maintaining persistent reusable backend socket pools.
- **Traffic Shaping & Security:** Per-client-IP token-bucket rate limiting and TLS termination.
- **Observability & Operational Visibility:** Prometheus metrics, structured JSON logging, and a real-time web dashboard.
- **Graceful Lifecycle Management:** Draining in-flight connections on shutdown signals (`SIGINT`/`SIGTERM`).
- **Benchmark-Driven Engineering:** Empirical performance benchmarking comparing pooling vs non-pooling strategies.

---

## ⚡ Current Features

| Feature | Description | Implementation Details |
|---|---|---|
| **Raw TCP Proxying** | Hand-written proxy loop accepting and forwarding raw socket streams | `net.Listen`, `tls.Listen`, `net.Dial` |
| **HTTP Request Parsing** | Extracts methods, URIs, headers, and streams request bodies | `readRequest`, `ExtractOrGenerateRequestID`, `ExtractCacheKey` |
| **Request Body Support** | Streams arbitrary request payloads (POST, PUT, JSON, forms) | `io.ReadFull` driven by `Content-Length` header |
| **Least-Connections Balancing** | Dynamically routes traffic to healthy backends with fewest active connections | `getLeastConnectionsBackend()` with mutex safety |
| **Per-Backend Circuit Breaker** | State machine (`CLOSED` → `OPEN` → `HALF-OPEN` → `CLOSED`) preventing cascade failures | `circuit_breaker.go` with failure thresholds & cooldown |
| **Correlation Request IDs** | Generates/propagates `X-Request-ID` across client responses, logs, and backends | `request_id.go` with `crypto/rand` UUID generation |
| **Structured Logging** | Production-grade operational logging with context attributes | `logger.go` using Go 1.21+ `log/slog` (JSON & Text) |
| **In-Memory Response Cache** | Thread-safe GET `200 OK` response cache with TTL expiration | `cache.go` using `sync.RWMutex` |
| **Production Hardening** | Per-connection panic recovery, socket read/write deadlines, and max body size limits | `hardening.go` (`RecoverConnection`, `SetConnTimeouts`) |
| **Active Health Checking** | Background goroutine probing backends at configurable intervals | `healthCheck()` with automatic node removal & recovery |
| **Connection Pooling** | Reuses persistent TCP connections per backend to minimize socket creation | `ConnPool` with 10 max idle connections & 30s idle TTL |
| **Token-Bucket Rate Limiting** | Limits request frequency per client IP (5 burst, 1 token/sec refill) | `ratelimit.go` returning compliant `429` responses |
| **Prometheus Metrics** | Plain-text metrics server exposing request, error, cache, and breaker counters | `metrics.go` served on `:9090/metrics` |
| **Web Observability Dashboard** | Real-time glassmorphic monitoring dashboard and JSON status API | `dashboard.go` served on `:9090/dashboard` & `/api/status` |
| **TLS Termination** | Terminates HTTPS TLS connections using X.509 certificates | `crypto/tls` with `cert.pem` and `key.pem` |
| **Graceful Shutdown** | Intercepts `SIGINT`/`SIGTERM`, stops listener, and drains in-flight requests | `sync.WaitGroup` tracking active handlers in `main.go` |
| **Kubernetes Readiness** | Dedicated `/healthz` (liveness) and `/readyz` (readiness) endpoints | `k8s_probes_test.go` and `k8s/` production manifests |
| **Containerization & CI/CD** | Multi-stage Docker definitions, Docker Compose, and GitHub Actions pipeline | `Dockerfile`, `docker-compose.yml`, `.github/workflows/ci.yml` |

---

## 🏗️ Architecture

```
                       +-----------------------------------+
                       |    Client Requests (HTTPS/TLS)    |
                       +-----------------------------------+
                                         |
                                         v
                       +-----------------------------------+
                       | TLS Termination & Panic Recovery  |
                       +-----------------------------------+
                                         |
                                         v
                       +-----------------------------------+
                       | Per-Client Token Bucket Limiter   |
                       +-----------------------------------+
                                         |
                                         v
                       +-----------------------------------+
                       | In-Memory Response Cache (GET 200)|
                       +-----------------------------------+
                                         |
                                         v
                       +-----------------------------------+
                       | Least-Connections & Circuit       |
                       | Breaker Backend Router            |
                       +-----------------------------------+
                                         |
                       +-----------------------------------+
                       | Persistent Connection Pool (30s)  |
                       +-----------------------------------+
                       /                 |                 \
                      v                  v                  v
           +------------------+ +------------------+ +------------------+
           | Backend 1 (:9001)| | Backend 2 (:9002)| | Backend 3 (:9003)|
           +------------------+ +------------------+ +------------------+
                      ^                  ^                  ^
                      |                  |                  |
           +--------------------------------------------------------+
           | Background Health Checker Goroutine (5s Probe Interval)|
           +--------------------------------------------------------+

   [ Admin & Observability Server (:9090) ]
   ├── /dashboard   --> Real-time Glassmorphic Web Dashboard
   ├── /api/status  --> JSON Cluster Health & Performance API
   ├── /metrics     --> Prometheus Metrics Endpoint
   ├── /healthz     --> Kubernetes Liveness Probe
   └── /readyz      --> Kubernetes Readiness Probe
```

---

## 🔄 Request Lifecycle

1. **Connection & TLS:** Client connects to port `:8080` over HTTPS. The proxy terminates TLS via `tls.Listen` using `cert.pem` and `key.pem`.
2. **Panic Guard & Timeouts:** Connection handler defers `RecoverConnection()` to trap panics, and applies read/write socket deadlines (`SetConnTimeouts`).
3. **Rate Limit Evaluation:** Client IP (`net.SplitHostPort`) is checked against `RateLimiter.Allow()`. If bucket tokens $< 1.0$, returns HTTP `429 Too Many Requests` with `X-Request-ID`.
4. **Header & Body Parsing:** Raw HTTP request is parsed (`readRequest`). `X-Request-ID` is extracted or generated using `crypto/rand`.
5. **Response Cache Lookup:** For `GET` requests, `ExtractCacheKey` queries `globalCache.Get()`. On hit, returns cached response immediately.
6. **Backend Selection:** On cache miss, `getLeastConnectionsBackend()` selects the healthy backend with minimum active connections whose Circuit Breaker is not `OPEN`.
7. **Active Connection Tracking:** Active connection count for target backend is incremented under mutex lock; deferred decrement is registered.
8. **Connection Pool Retrieval:** `ConnPool.Get()` fetches an idle persistent connection (verifying idle age $\le 30\text{s}$). Dials fresh TCP connection if pool is empty or stale.
9. **Forwarding & Execution:** Request payload is forwarded to backend. Response is parsed via `http.ReadResponse`, and `X-Request-ID` is injected.
10. **Caching & Client Response:** If response is `200 OK` from `GET`, response bytes are saved in `globalCache`. Response is written back to client.
11. **Connection Return & Metrics:** Backend connection is returned to `ConnPool` (or closed if pool capacity reached). Circuit Breaker success/failure is recorded, and Prometheus metrics are updated.

---

## ⚖️ Load Balancing Algorithms

### 1. Least-Connections (Default)
Tracks currently in-flight requests per backend (`ActiveConnections`). When a new request arrives, `getLeastConnectionsBackend()` locks the backend list, filters for healthy nodes with active circuit breakers, and selects the node handling the fewest concurrent connections.

*Why Least-Connections?* Unlike Round-Robin, Least-Connections dynamically adapts to heterogeneous backends or uneven request processing times. If Backend 1 is executing a slow database query, subsequent incoming requests automatically bypass Backend 1 and route to faster, idle backends.

### 2. Round-Robin (Fallback Baseline)
Sequential distribution iterating through backends in cyclical order (`counter % len(backends)`), skipping unhealthy nodes.

---

## 🩺 Health Checks & Failover

- **Active Probing:** A background goroutine (`healthCheck`) runs periodically based on `health_check_interval_seconds` in `config.yaml`.
- **Automatic Failover:** Sends plain HTTP `GET /` probe requests to backends. If a backend fails to respond or returns an error, `backend.Healthy` is set to `false`. `getLeastConnectionsBackend()` immediately excludes unhealthy backends from routing.
- **Automatic Recovery:** When an unhealthy backend resumes responding cleanly to health probes, `backend.Healthy` is reset to `true` and it automatically re-enters traffic rotation.

---

## 🛑 Per-Backend Circuit Breaker

To prevent cascading failures when a backend suffers degradation:
- **State Machine:** `CLOSED` (Normal) → `OPEN` (Tripped) → `HALF-OPEN` (Trialing) → `CLOSED`.
- **Failure Tracking:** Increments consecutive failure count when socket errors occur during request forwarding.
- **Tripping:** Reaching `circuit_breaker_threshold` consecutive failures immediately trips breaker to `OPEN`, failing fast without hitting the degraded backend.
- **Cooldown & Recovery:** After `circuit_breaker_cooldown_seconds`, transitions to `HALF-OPEN` and permits a single trial request. Success resets state to `CLOSED`; failure re-trips to `OPEN`.

---

## ⚡ Connection Pooling & Benchmark Findings

### Mechanics
Creating a new TCP socket per request involves a 3-way handshake (`SYN`, `SYN-ACK`, `ACK`) and socket allocation overhead. `ConnPool` maintains a thread-safe pool (`map[string][]pooledConn`) of persistent backend sockets.
- **Idle Expiration:** Connections idle for $> 30\text{s}$ are closed and discarded.
- **Pool Capacity:** Maximum 10 idle connections stored per backend.

### Microbenchmark Results (`benchmark_test.go`)
Executed with `go test -bench=BenchmarkProxy -benchmem`:

| Benchmark Variant | Latency (ns/op) | Memory Allocated (B/op) | Allocations (allocs/op) |
|---|---|---|---|
| **With Connection Pooling** | **1,583,451 ns** (~1.58 ms) | **2,001 B** | **17 allocs** |
| Direct Dialing (No Pooling) | 2,349,063 ns (~2.35 ms) | 4,554 B | 55 allocs |
| **Performance Difference** | **~33% Lower Latency** | **~56% Memory Savings** | **~70% Fewer Allocs** |

*Analysis:* Reusing persistent TCP connections eliminates recurring socket handshakes and reduces garbage collection pressure by 70%.

---

## 🛡️ Per-Client-IP Rate Limiting

Implemented in [`ratelimit.go`](file:///d:/reverse-proxy/ratelimit.go) using a Token Bucket algorithm per client IP (`ExtractIP`):
- **Capacity:** 5 burst tokens maximum.
- **Refill Rate:** 1 token refill per second.
- **Behavior:** Each request consumes 1 token. When empty, requests are rejected immediately with HTTP `429 Too Many Requests` and `Connection: close` headers.

---

## 📊 Metrics & Observability Dashboard

### Prometheus Metrics Endpoint (`:9090/metrics`)
Exposes operational metrics in standard Prometheus plain-text exposition format:
```text
# HELP proxy_requests_total Total number of HTTP requests processed by the proxy.
# TYPE proxy_requests_total counter
proxy_requests_total 142

# HELP proxy_errors_total Total number of HTTP request errors encountered by the proxy.
# TYPE proxy_errors_total counter
proxy_errors_total 2

# HELP proxy_backend_requests_total Total number of requests sent to each backend.
# TYPE proxy_backend_requests_total counter
proxy_backend_requests_total{backend="localhost:9001"} 48
proxy_backend_requests_total{backend="localhost:9002"} 47
proxy_backend_requests_total{backend="localhost:9003"} 45

# HELP proxy_circuit_breaker_state Current state of backend circuit breaker (1 for active state).
# TYPE proxy_circuit_breaker_state gauge
proxy_circuit_breaker_state{backend="localhost:9001",state="CLOSED"} 1

# HELP proxy_cache_hits_total Total response cache hits.
# TYPE proxy_cache_hits_total counter
proxy_cache_hits_total 34
```

### Live Web Dashboard (`:9090/dashboard`)
A glassmorphic real-time UI built with HTML/CSS/JS that polls `/api/status` every 2 seconds displaying total requests, error rates, cache hit ratios, backend health badges, active connection gauges, and circuit breaker states.

---

## 🛑 Graceful Shutdown

Upon receiving `SIGINT` (Ctrl+C) or `SIGTERM`:
1. `shutdownChan` catches the OS signal.
2. `isShuttingDown` flag is set under `shutdownMu` lock.
3. Listener is closed (`listener.Close()`), stopping acceptance of new TCP connections.
4. `/readyz` probe immediately returns `503 Service Unavailable`.
5. Main thread executes `wg.Wait()`, waiting for all active `handleConnection` goroutines to finish.
6. Console prints `"All requests completed. Shutting down cleanly."` and exits with code 0.

---

## 🔒 TLS Termination

```
Client  --- [ HTTPS / TLS Encrypted ] --->  Reverse Proxy (:8080)
                                                    |
                                            (TLS Termination)
                                                    |
Proxy   --- [ Plain HTTP / TCP ] --------->  Backend Servers (:9001, :9002, :9003)
```
- Proxy accepts secure client traffic via `tls.Listen("tcp", addr, tlsConfig)`.
- TLS certificates loaded via `tls.LoadX509KeyPair("cert.pem", "key.pem")`.
- Backend communication remains lightweight plain HTTP over local/private network.
- *Security Note:* `key.pem` is private and excluded from Git tracking via `.gitignore`. Self-signed certificates require `-k` (insecure flag) when testing with `curl`.

---

## 🛠️ Configuration (`config.yaml`)

```yaml
port: 8080
health_check_interval_seconds: 5
circuit_breaker_threshold: 3
circuit_breaker_cooldown_seconds: 5
log_level: "INFO"
log_json: true
cache_enabled: true
cache_ttl_seconds: 30
read_timeout_seconds: 10
write_timeout_seconds: 10
max_body_bytes: 10485760
backends:
  - localhost:9001
  - localhost:9002
  - localhost:9003
```

All ports, timeouts, thresholds, and backend targets can be altered without recompiling Go source code.

---

## 💻 Tech Stack

- **Language:** Go 1.21+
- **Networking:** `net` (`net.Listen`, `net.Dial`, TCP Sockets)
- **Security:** `crypto/tls` (X.509 TLS Termination)
- **Structured Logging:** `log/slog` (Go 1.21+ Standard Library)
- **Configuration:** `gopkg.in/yaml.v3`
- **Concurrency:** Goroutines, `sync.Mutex`, `sync.RWMutex`, `sync.WaitGroup`
- **Containerization:** Docker, Docker Compose
- **Orchestration:** Kubernetes Manifests (`k8s/`) & Health Probes (`/healthz`, `/readyz`)
- **CI/CD:** GitHub Actions (`.github/workflows/ci.yml`)

---

## 📁 Project Structure

```
d:\reverse-proxy
├── .github/workflows/
│   └── ci.yml             # GitHub Actions CI workflow (gofmt, vet, test -race, docker)
├── backends/              # Upstream HTTP backend test servers
│   ├── backend1.go        # Backend 1 (Port 9001)
│   ├── backend2.go        # Backend 2 (Port 9002)
│   └── backend3.go        # Backend 3 (Port 9003)
├── k8s/                   # Production Kubernetes manifests
│   ├── configmap.yaml     # Kubernetes ConfigMap for config.yaml
│   ├── deployment.yaml    # Kubernetes Deployment manifest with probes & limits
│   └── service.yaml       # Kubernetes Service manifest (LoadBalancer)
├── benchmark_test.go      # Go benchmark suite for connection pooling evaluation
├── cache.go               # In-memory response cache engine
├── cache_test.go          # Unit tests for response caching
├── circuit_breaker.go     # Circuit breaker state machine implementation
├── circuit_breaker_test.go# Unit tests for circuit breaker state transitions
├── config.go              # YAML configuration loader
├── config.yaml            # Proxy configuration file
├── config_test.go         # Unit tests for configuration parsing
├── conn_pool_test.go      # Unit tests for connection pool mechanics
├── dashboard.go           # Web observability dashboard & status API handlers
├── dashboard_test.go      # Unit tests for dashboard endpoints
├── Dockerfile             # Multi-stage Dockerfile for reverse proxy binary
├── Dockerfile.backend     # Multi-stage Dockerfile for backend servers
├── docker-compose.yml     # Docker Compose orchestration manifest
├── hardening.go           # Panic recovery handler and socket timeout setters
├── hardening_test.go      # Unit tests for panic recovery and timeouts
├── integration_test.go    # End-to-end integration tests using real TCP servers
├── k8s_probes_test.go     # Unit tests for /healthz and /readyz probes
├── logger.go              # Structured logging initializer using log/slog
├── logger_test.go         # Unit tests for structured logger
├── main.go                # Core raw TCP reverse proxy engine entrypoint
├── metrics.go             # Prometheus metrics exporter implementation
├── ratelimit.go           # Token-bucket per-IP rate limiter
├── ratelimit_test.go      # Unit tests for token bucket rate limiting
├── request_id.go          # X-Request-ID extraction and generation engine
├── request_id_test.go     # Unit tests for request correlation IDs
├── run_all.bat            # Windows Terminal multi-pane launch script
├── run.ps1                # PowerShell launcher script
├── run.sh                 # Linux / macOS launcher script
├── cert.pem / key.pem     # TLS certificate & private key (key.pem git-ignored)
└── README.md              # Project documentation
```

---

## 🚀 Getting Started

### 1. Clone Repository
```bash
git clone https://github.com/sujayghosh13/go-reverse-proxy.git
cd go-reverse-proxy
```

### 2. Generate Local Self-Signed TLS Certificates
```bash
openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=localhost"
```

### 3. Launch Application Suite

**Option A (Automated Launch Script):**
```cmd
run_all.bat
```

**Option B (Manual Startup):**
```bash
# Terminal 1, 2, 3: Start Backends
go run backends/backend1.go
go run backends/backend2.go
go run backends/backend3.go

# Terminal 4: Start Proxy
go run .
```

### 4. Verify & Test Endpoints

```bash
# Test HTTPS GET (use -k because certificate is self-signed for localhost)
curl -k https://localhost:8080/

# Test HTTPS POST with Request Body Payload
curl -k -X POST -d "Hello Reverse Proxy" https://localhost:8080/

# Test Request Correlation ID Propagation
curl -k -i -H "X-Request-ID: test-custom-id-123" https://localhost:8080/

# View Prometheus Metrics
curl http://localhost:9090/metrics

# View Kubernetes Health Probes
curl http://localhost:9090/healthz
curl http://localhost:9090/readyz

# Open Web Observability Dashboard in Browser
# http://localhost:9090/dashboard
```

---

## 🧪 Manual & Automated Testing

### Automated Test Suite Execution
Run all unit and integration tests with Go's data race detector:
```bash
go test -v -race ./...
```

### Manual Behavior Verification
1. **TLS Certificate Enforcement:** Running `curl https://localhost:8080/` without `-k` fails with a certificate verification error as expected for self-signed keys; adding `-k` succeeds cleanly.
2. **Rate Limiting Rejection:** Firing 10 rapid sequential requests triggers HTTP `429 Too Many Requests` after the 5-token burst capacity is exhausted.
3. **Least-Connections Routing:** Requests distribute dynamically across backends based on current active in-flight request counts.
4. **Graceful Shutdown Wait:** Sending a `SIGINT` (Ctrl+C) while an in-flight request is executing displays `"Shutdown signal received. Finishing in-flight requests..."` and waits for completion before printing `"All requests completed. Shutting down cleanly."` and exiting.
5. **Panic Isolation:** Artificially triggering a panic inside a handler triggers `RecoverConnection()`, logs a stacktrace, writes an HTTP `500 Internal Server Error` to the client, and keeps the proxy process alive for other requests.

---

## 💥 Fault Tolerance Verification

```
Backend Crashes / Network Disconnection
                 │
                 ▼
Active Health Probe / Circuit Breaker detects failure
                 │
                 ▼
Backend marked Unhealthy / Circuit Breaker transitions to OPEN
                 │
                 ▼
Proxy routes 100% of traffic to remaining healthy backends
                 │
                 ▼
Backend restarts / recovers
                 │
                 ▼
Health Probe succeeds / Circuit Breaker enters HALF-OPEN trial
                 │
                 ▼
Trial succeeds -> Breaker resets to CLOSED -> Restored to cluster rotation
```

---

## 📐 Design Decisions & Tradeoffs

- **Why Raw TCP Sockets vs `net/http` Proxy?** Hand-writing the socket loop provides complete visibility into buffer management, HTTP header framing, socket lifecycle mechanics, and protocol nuances.
- **Why Least-Connections over Round-Robin?** Round-robin assumes all requests require equal processing time. Least-connections adapts dynamically when individual backends become bottlenecked by expensive operations.
- **Why Circuit Breakers in addition to Health Checks?** Periodic health checks run every 5 seconds. If a backend crashes between probes, incoming requests would fail for up to 5 seconds. The Circuit Breaker trips immediately upon detecting consecutive inline forwarding errors.
- **Why Connection Pooling?** Dialing new TCP connections per request introduces latency and ephemeral port exhaustion under heavy traffic. Persistent socket pooling reduces latency by ~33%.
- **Why Process-Local Pooling & Caching?** Eliminates external dependencies (like Redis) for lightweight standalone binary deployments.

---

## ⚠️ Known Limitations

- **Localhost Testing Certificates:** Uses self-signed X.509 certificates (`cert.pem`/`key.pem`) intended for local testing and development.
- **Unencrypted Internal Backend Traffic:** Terminates TLS at the proxy edge; communication between proxy and backend servers remains plain HTTP over the internal network (standard reverse proxy edge architecture).
- **Process-Local State:** Connection pool, rate limiter, circuit breakers, and response cache are maintained in-memory per proxy instance.
- **Single-Host Benchmarks:** Localhost benchmark numbers reflect single-machine loopback performance where TCP handshake costs are minimal compared to real WAN networks.

---

## 🔮 Production Roadmap

Future architecture enhancements under consideration:
- [ ] **Distributed Cache & Rate Limiting:** External Redis backed state for multi-instance proxy clusters.
- [ ] **gRPC & HTTP/2 Multiplexing:** Frame parsing and stream multiplexing for gRPC payloads.
- [ ] **Dynamic Configuration Reloading:** Hot-reloading `config.yaml` on `SIGHUP` without restarting the process.
- [ ] **eBPF Socket Acceleration:** Kernel-level socket steering for ultra-low latency routing.

---

## 💡 What This Project Demonstrates

- **Low-Level Socket Programming:** Socket creation, listener loops, stream buffering, deadline control.
- **HTTP Protocol Fundamentals:** Manual header parsing, body streaming, HTTP response code framing.
- **Concurrent Programming in Go:** Goroutines, channels, `sync.Mutex`, `sync.RWMutex`, `sync.WaitGroup`, data race elimination.
- **Distributed Resilience Patterns:** Least-connections load balancing, token bucket rate limiting, circuit breaker state machines, connection pooling.
- **Cloud-Native Engineering:** Docker containerization, Kubernetes readiness probes, Prometheus metrics, structured logging.
- **Empirical System Evaluation:** Microbenchmarking, memory allocation profiling, fault tolerance verification.

---

## 📝 Summary

> **Go Reverse Proxy & Load Balancer:** Designed and built a high-performance, resilient HTTP/HTTPS reverse proxy and load balancer in Go from raw TCP sockets (`net.Listen`, `net.Dial`, `crypto/tls`) without relying on standard `net/http` proxy abstractions. Implemented least-connections load balancing, per-backend circuit breaking (`CLOSED`/`OPEN`/`HALF-OPEN`), token-bucket rate limiting, in-memory response caching, TLS termination, and persistent TCP connection pooling. Built for cloud-native operational standards with Go 1.21 `log/slog` structured logging, Prometheus metrics, a real-time web dashboard, Kubernetes liveness/readiness probes, Docker Compose containerization, and a GitHub Actions CI pipeline. Verified via comprehensive automated unit/integration tests (`go test -race`) and microbenchmarks demonstrating a ~33% latency reduction through socket reuse.

---

## 🧭 Project Philosophy

```
  Networking (TCP Sockets)
            │
            ▼
  HTTP Protocol Parsing
            │
            ▼
  Concurrency & Synchronization
            │
            ▼
  Load Balancing & Routing
            │
            ▼
  Health Probes & Circuit Breaking
            │
            ▼
  Connection Pooling & Caching
            │
            ▼
  Security (TLS Termination & Rate Limiting)
            │
            ▼
  Observability & Metrics
            │
            ▼
  Reliability & Performance Verification
```

*This project is dedicated to engineering depth, systems understanding, and empirical verification over superficial abstractions.*
