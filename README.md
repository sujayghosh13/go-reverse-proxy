# Go Reverse Proxy & Load Balancer (Built from Scratch)

A high-performance reverse proxy and load balancer built from raw TCP sockets in Go — no `net/http` framework used for the proxy listener/forwarding core. Built to understand exactly how tools like Nginx and HAProxy work under the hood.

## Key Features

- **Raw TCP Socket Processing:** Accepts incoming client HTTP requests over raw TCP listeners (`net.Listen`) and parses HTTP headers manually.
- **Connection Pooling & Idle Cleanup (`ConnPool`):** Maintains an in-memory thread-safe connection pool of persistent backend TCP sockets (`pooledConn`). Limits idle sockets to 10 per backend and automatically closes/discards connections idle for $> 30\text{s}$.
- **HTTP Request Body Support:** Parses `Content-Length` headers and streams exact body bytes using `io.ReadFull` for GET, POST, PUT, JSON, and Form payloads.
- **YAML Config File Support (`config.yaml`):** Externalized configuration for proxy port, health check probe intervals, and backend server target lists using `gopkg.in/yaml.v3`.
- **Prometheus Metrics Endpoint (`:9090/metrics`):** Runs a separate `net/http` server on port 9090 tracking total requests, error rates, and per-backend traffic counters in Prometheus plain-text format.
- **Round-Robin Load Balancing:** Evenly distributes client traffic across active backends using a thread-safe mutex counter.
- **Active Health Checks & Failover:** Periodically pings backends on a configurable interval; automatically removes unhealthy nodes from rotation and restores them upon recovery.
- **Resilient Fallbacks:** Gracefully handles stale/closed pooled connections with automatic fresh socket reconnect retries and returns `503 Service Unavailable` if all backends are down.
- **High Concurrency:** Concurrent request handling powered by Go goroutines and `sync.Mutex` synchronization.

## Architecture

```
Client  --->  Reverse Proxy (Configured Port, e.g. :8080)  --->  Backend 1 (:9001)
                     |                                              
                     |--- Round-Robin / Connection Pool --------->  Backend 2 (:9002)
                     |                                              
                     |--- Active Health Checks ------------------>  Backend 3 (:9003)
                     |
                     v
             Metrics Endpoint (:9090/metrics)
```

1. Client connects to the proxy on port configured in `config.yaml` (default `:8080`).
2. Proxy parses incoming HTTP headers line-by-line and extracts `Content-Length` for request body payloads.
3. `getNextBackend()` selects the next **healthy** backend in rotation (round-robin).
4. Proxy retrieves an idle TCP connection from `ConnPool` (verifying connection is $\le 30\text{s}$ old; dials a fresh connection if empty/stale).
5. Request and body payload are forwarded to the backend.
6. Backend response is parsed (`http.ReadResponse`) and written to the client (`resp.Write`).
7. Connection is returned to `ConnPool` (if pool count $< 10$; otherwise connection is closed).
8. `globalMetrics.RecordRequest()` records success/error metrics.
9. Background `healthCheck` goroutine pings backends on the interval specified in `config.yaml`.

## Configuration (`config.yaml`)

Edit `config.yaml` to change proxy port, health check probe frequency, or backend addresses without recompiling:

```yaml
port: 8080
health_check_interval_seconds: 5
backends:
  - localhost:9001
  - localhost:9002
  - localhost:9003
```

## Observability & Metrics (`:9090/metrics`)

The proxy exposes a Prometheus-compatible metrics endpoint on port **`9090`**:

```bash
curl http://localhost:9090/metrics
```

**Sample Output:**
```text
# HELP proxy_requests_total Total number of HTTP requests processed by the proxy.
# TYPE proxy_requests_total counter
proxy_requests_total 12

# HELP proxy_errors_total Total number of HTTP request errors encountered by the proxy.
# TYPE proxy_errors_total counter
proxy_errors_total 0

# HELP proxy_backend_requests_total Total number of requests sent to each backend.
# TYPE proxy_backend_requests_total counter
proxy_backend_requests_total{backend="localhost:9001"} 4
proxy_backend_requests_total{backend="localhost:9002"} 4
proxy_backend_requests_total{backend="localhost:9003"} 4
```

## Tech Stack

- **Language:** Go 1.26+
- **Concurrency:** Goroutines & `sync.Mutex`
- **Config Parser:** `gopkg.in/yaml.v3`
- **Networking:** `net.Listen`, `net.Dial`, `bufio`, `io`, `net/http`

## Running Locally

### Option 1: Automated Launch Scripts

**Windows Terminal / Batch:**
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

### Option 2: Manual Terminal Startup

1. Start each backend in separate terminals:
```bash
go run backends/backend1.go   # port 9001
go run backends/backend2.go   # port 9002
go run backends/backend3.go   # port 9003
```

2. Start the reverse proxy:
```bash
go run .
```

3. Test proxy traffic & metrics:
```bash
# Proxy HTTP GET
curl http://localhost:8080/

# Proxy HTTP POST with Body
curl -X POST -d "Hello Reverse Proxy" http://localhost:8080/

# Fetch Prometheus Metrics
curl http://localhost:9090/metrics
```

## Benchmarks

Tested with [`hey`](https://github.com/rakyll/hey), 2000 requests at 50 concurrent connections:

| Metric | Through Proxy (:8080) | Direct to Backend (:9001) |
|---|---|---|
| Requests/sec | ~2,119 | ~16,553 |
| Average latency | 23.2 ms | 2.8 ms |
| p99 latency | 53.2 ms | 40.3 ms |
| Success rate | 100% (2000/2000) | 100% (2000/2000) |

**Connection Pooling:** Built-in connection pooling (`ConnPool`) manages reusable persistent backend connections (max 10 per backend, 30s max idle TTL), avoiding repeated TCP setup and teardown overhead on sequential requests.

**Fault tolerance verified:** Killing a backend mid-traffic causes it to be excluded from rotation within one health-check cycle (≤5s), with zero failed client requests. Restarting it automatically restores it to rotation.

## Features & Roadmap

- [x] **Connection pooling** to backends (`ConnPool` with 10 idle limits & 30s TTL cleanup)
- [x] **Support for HTTP request bodies** (`Content-Length` detection & `io.ReadFull` payload streaming)
- [x] **YAML Config File Support** (`config.yaml` + `config.go` for port, health check interval, and backends)
- [x] **Structured metrics endpoint (`/metrics`) for observability** (Prometheus plain-text server on port 9090)
- [ ] Weighted round-robin / least-connections load balancing strategies
- [ ] TLS termination

## What this project demonstrates

- Raw socket programming and manual HTTP parsing
- Connection pooling design with idle connection expiration & capacity management
- Concurrent programming in Go (goroutines, mutexes, race condition prevention)
- Load balancing and failover design
- Externalized application configuration management
- Observability & metrics collection in Prometheus format
- Performance benchmarking and system architecture evaluation
