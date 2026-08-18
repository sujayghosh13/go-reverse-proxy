# Go Reverse Proxy & Load Balancer (Built from Scratch)

A reverse proxy and load balancer built from raw TCP sockets in Go — no `net/http` framework used for the proxy itself. Built to understand exactly how tools like nginx and HAProxy work under the hood, rather than wrapping one.

## What it does

- Accepts incoming HTTP requests over a raw TCP listener and parses them manually (no framework)
- Distributes traffic across multiple backend servers using round-robin load balancing
- Runs background health checks every 5 seconds; automatically routes around backends that go down and reintroduces them when they recover
- Returns a proper `503 Service Unavailable` if every backend is down, instead of hanging or crashing
- Handles many simultaneous connections concurrently using goroutines

## Why I built it this way

Instead of using Go's `net/http` package (which hides almost everything interesting), I built the proxy directly on top of `net.Listen` / `net.Dial` and parsed HTTP as raw text. The goal was to understand:
- What an HTTP request actually looks like on the wire
- How a reverse proxy is simultaneously a server (to the client) and a client (to the backend)
- Why concurrency, locking, and health checking matter in real load balancers

## Architecture

```
Client  --->  Reverse Proxy (:8080)  --->  Backend 1 (:9001)
                     |
                     |--- round-robin --->  Backend 2 (:9002)
                     |
                     |--- health checks -->  Backend 3 (:9003)
```

1. Client connects to the proxy on port 8080
2. Proxy reads the raw HTTP request off the TCP connection
3. `getNextBackend()` selects the next **healthy** backend in rotation (round-robin, protected by a mutex for concurrency safety)
4. Proxy opens a new TCP connection to that backend and forwards the request
5. Backend's response is streamed straight back to the client via `io.Copy`
6. A background goroutine independently pings all backends every 5 seconds, marking them healthy/unhealthy

## Tech

- **Language:** Go (chosen for lightweight concurrency via goroutines and built-in `net` package)
- **Concurrency:** each connection handled in its own goroutine; a mutex protects the shared round-robin counter and backend health state
- **No external dependencies** for the proxy logic itself

## Running it locally

Start each backend in its own terminal:
```bash
cd backends
go run backend1.go   # port 9001
go run backend2.go   # port 9002
go run backend3.go   # port 9003
```

Start the proxy:
```bash
go run main.go        # port 8080
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

**Why the overhead exists:** the proxy currently opens a brand-new TCP connection to the backend for every request instead of reusing a pooled connection — this is the standard optimization tools like nginx use, and is the top candidate for future improvement. The remaining gap comes from the extra network hop (client → proxy → backend → proxy → client) and manual line-by-line HTTP parsing.

**Fault tolerance verified:** killing a backend mid-traffic causes it to be excluded from rotation within one health-check cycle (≤5s), with zero failed client requests. Restarting it automatically restores it to rotation.

## What I'd improve next

- **Connection pooling** to backends (biggest expected performance win)
- Support for HTTP request bodies (currently only handles headers, e.g. GET requests)
- Weighted round-robin / least-connections load balancing strategies
- Config file for backend list instead of hardcoding
- TLS termination
- Structured metrics endpoint (`/metrics`) for observability

## What this project demonstrates

- Raw socket programming and manual HTTP parsing
- Concurrent programming in Go (goroutines, mutexes, race condition prevention)
- Load balancing and failover design
- Performance benchmarking and the ability to explain *why* results look the way they do, not just report them
