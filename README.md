# Go Reverse Proxy & Load Balancer (Built from Scratch)

A reverse proxy and load balancer built from raw TCP sockets in Go — no `net/http` framework used for the proxy itself. Built to understand exactly how tools like nginx and HAProxy work under the hood, rather than wrapping one.

## What it does

- Accepts incoming HTTP requests over a raw TCP listener and parses them manually (no framework)
- Reuses TCP backend connections using an in-memory **Connection Pool** (`ConnPool`) to eliminate per-request TCP handshake overhead
- Distributes traffic across multiple backend servers using round-robin load balancing
- Runs background health checks every 5 seconds; automatically routes around backends that go down and reintroduces them when they recover
- Returns a proper `503 Service Unavailable` if every backend is down, instead of hanging or crashing
- Handles many simultaneous connections concurrently using goroutines
- Provides structured HTTP request logging and startup error handling across all backend servers

## Why I built it this way

Instead of using Go's `net/http` package (which hides almost everything interesting), I built the proxy directly on top of `net.Listen` / `net.Dial` and parsed HTTP as raw text. The goal was to understand:
- What an HTTP request actually looks like on the wire
- How a reverse proxy is simultaneously a server (to the client) and a client (to the backend)
- Why connection pooling, concurrency, locking, and health checking matter in real load balancers

## Architecture

```
Client  --->  Reverse Proxy (:8080)  --->  Backend 1 (:9001)
                     |
                     |--- round-robin --->  Backend 2 (:9002)
                     |
                     |--- health checks -->  Backend 3 (:9003)
```

1. Client connects to the proxy on port 8080.
2. Proxy reads the raw HTTP request off the client TCP connection.
3. `getNextBackend()` selects the next **healthy** backend in rotation (round-robin, protected by a mutex for concurrency safety).
4. Proxy requests a reusable TCP connection from `ConnPool` (or opens a fresh TCP connection if the pool is empty).
5. Request is forwarded to the selected backend.
6. Backend response is parsed (`http.ReadResponse`) and written to the client (`resp.Write`).
7. Backend connection is returned to `ConnPool` for future reuse (`pool.Put`).
8. A background goroutine independently pings all backends every 5 seconds, marking them healthy/unhealthy.

## Tech

- **Language:** Go (chosen for lightweight concurrency via goroutines and built-in `net` package)
- **Concurrency:** Each connection handled in its own goroutine; mutexes protect the connection pool, shared round-robin counter, and backend health state
- **No external dependencies** for the proxy logic itself

## Running it locally

### Option 1: Automated Launch Scripts

**Windows (Batch / Windows Terminal):**
```cmd
run_all.bat
```
*(or run `run_backends.bat` for backends only)*

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

Start each backend in its own terminal:
```bash
go run backends/backend1.go   # port 9001
go run backends/backend2.go   # port 9002
go run backends/backend3.go   # port 9003
```

Start the proxy:
```bash
go run main.go                # port 8080
```

Test it:
```bash
curl http://localhost:8080
```

## Benchmarks

Tested with [`hey`](https://github.com/rakyll/hey), 2000 requests at 50 concurrent connections:

| Metric | Through Proxy (:8080) | Direct to Backend (:9001) |
|---|---|---|
| Requests/sec | ~2,119 | ~16,553 |
| Average latency | 23.2 ms | 2.8 ms |
| p99 latency | 53.2 ms | 40.3 ms |
| Success rate | 100% (2000/2000) | 100% (2000/2000) |

**Connection Pooling:** Built-in connection pooling (`ConnPool`) manages reusable persistent backend connections, avoiding repeated TCP connection setup and teardown overhead on sequential requests.

**Fault tolerance verified:** Killing a backend mid-traffic causes it to be excluded from rotation within one health-check cycle (≤5s), with zero failed client requests. Restarting it automatically restores it to rotation.

## What I'd improve next

- [x] **Connection pooling** to backends (implemented)
- [ ] Support for HTTP request bodies (currently only handles headers, e.g. GET requests)
- [ ] Weighted round-robin / least-connections load balancing strategies
- [ ] Config file for backend list instead of hardcoding
- [ ] TLS termination
- [ ] Structured metrics endpoint (`/metrics`) for observability

## What this project demonstrates

- Raw socket programming and manual HTTP parsing
- Connection pooling design and mutex-based concurrency management
- Concurrent programming in Go (goroutines, mutexes, race condition prevention)
- Load balancing and failover design
- Performance benchmarking and system architecture evaluation
