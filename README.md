# High-Performance Production Go Reverse Proxy & Load Balancer

A production-grade, highly resilient reverse proxy and load balancer implemented from raw TCP sockets in Go — built without relying on standard `net/http` reverse-proxy abstractions (`httputil.ReverseProxy`) for the core proxy loop.

Engineered to demonstrate low-level networking, high-concurrency connection pooling, circuit breaking, distributed request correlation, structured observability, and cloud-native containerization.

---

## 🌟 Key Architecture & Capabilities

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
                       /                 |                 \
                      v                  v                  v
           +------------------+ +------------------+ +------------------+
           | Backend 1 (:9001)| | Backend 2 (:9002)| | Backend 3 (:9003)|
           +------------------+ +------------------+ +------------------+
                      ^                  ^                  ^
                      |                  |                  |
           +--------------------------------------------------------+
           | Persistent Connection Pool (30s Idle Cleanup)         |
           +--------------------------------------------------------+

   [ Admin / Metrics Server (:9090) ]
   ├── /dashboard   --> Real-time Glassmorphic Observability UI
   ├── /metrics     --> Prometheus Metrics Collector
   ├── /healthz     --> Kubernetes Liveness Probe
   └── /readyz      --> Kubernetes Readiness Probe
```

### Core Features
- 🔒 **TLS Termination:** Accepts HTTPS over TLS listeners (`tls.Listen`) with X.509 certificate pairs (`cert.pem`, `key.pem`).
- ⚡ **Connection Pooling (`ConnPool`):** Reuses persistent TCP sockets per backend with automatic 30s idle connection expiration and capacity management.
- ⚡ **In-Memory Response Cache:** Caches `200 OK` `GET` responses with configurable TTL and hit/miss tracking.
- ⚡ **Least-Connections Load Balancing:** Dynamically routes requests to healthy backends with minimum active connections.
- 🛑 **Per-Backend Circuit Breaker:** Full state machine (`CLOSED` → `OPEN` → `HALF-OPEN` → `CLOSED`) to prevent cascading failures.
- 🛡️ **Per-Client Token Bucket Rate Limiting:** Limits requests per client IP (5-token burst capacity, 1 token/sec refill) returning standard `429 Too Many Requests`.
- 🆔 **Request Correlation IDs:** Generates/propagates `X-Request-ID` across client responses, proxy logs, and backend headers.
- 📊 **Structured Logging (`log/slog`):** High-performance JSON/text structured operational logs with latency, status, backend, and error context.
- 🛡️ **Production Hardening:** Per-goroutine panic recovery, configurable read/write timeouts, and body/header size guardrails.
- 📊 **Observability Dashboard & Prometheus Metrics:** Live dark-mode web UI (`:9090/dashboard`) and Prometheus plain-text endpoint (`:9090/metrics`).
- 🛑 **Graceful Shutdown:** Catches `os.Interrupt`/`SIGTERM`, closes listener, drains in-flight requests via `sync.WaitGroup`, and cleanly terminates.
- 🐳 **Docker & Kubernetes Ready:** Multi-stage `Dockerfile`, `docker-compose.yml`, `/healthz`, `/readyz` probes, and complete `k8s/` manifests.

---

## 🚀 Quick Start & Running Locally

### 1. Automated Script Launchers

**Windows Terminal / Command Prompt:**
```cmd
run_all.bat
```

**PowerShell:**
```powershell
.\run.ps1
```

**Linux / macOS:**
```bash
chmod +x run.sh
./run.sh
```

### 2. Manual Local Execution

1. **Start Backend Servers:**
   ```bash
   go run backends/backend1.go   # Port 9001
   go run backends/backend2.go   # Port 9002
   go run backends/backend3.go   # Port 9003
   ```

2. **Generate TLS Certificates (Self-Signed):**
   ```bash
   openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=localhost"
   ```

3. **Launch Reverse Proxy:**
   ```bash
   go run .
   ```

4. **Verify HTTPS & Admin Services:**
   ```bash
   # HTTPS GET Request (with self-signed TLS bypass -k)
   curl -k https://localhost:8080/

   # View Live Web Dashboard
   # Open http://localhost:9090/dashboard in your browser

   # View Prometheus Metrics
   curl http://localhost:9090/metrics

   # Kubernetes Health Probes
   curl http://localhost:9090/healthz
   curl http://localhost:9090/readyz
   ```

---

## ⚙️ Configuration Guide (`config.yaml`)

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

---

## 📊 Benchmarks & Performance Metrics

Running `go test -bench=BenchmarkProxy -benchmem`:

| Implementation Strategy | Throughput / Latency | Memory Allocation | Allocs / Op |
|-------------------------|----------------------|-------------------|-------------|
| **With Connection Pooling** | **1.58 ms / op** | **2,001 B / op** | **17 allocs / op** |
| Without Connection Pooling (Fresh Dial) | 2.34 ms / op | 4,554 B / op | 55 allocs / op |
| **Performance Gain** | **~33% Lower Latency** | **~56% Less Memory** | **~70% Fewer Allocs** |

---

## 🐳 Containerization & Kubernetes

### Docker Compose
Run the proxy along with 3 backend replicas in an isolated container network:
```bash
docker-compose up --build
```

### Kubernetes Deployment
Deploy using production manifests in `k8s/`:
```bash
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

---

## 🧪 Automated Testing & CI/CD

Run full test suite with race detection:
```bash
go test -v -race ./...
```

Run code linter / vet check:
```bash
go vet ./...
```

Continuous Integration is configured via GitHub Actions (`.github/workflows/ci.yml`), validating code formatting (`gofmt`), static analysis (`go vet`), unit/integration test suite execution, and Docker build integrity.

---

## 📄 License & Author

Developed as an advanced systems engineering portfolio project. Built from scratch by Sujay Ghosh.
