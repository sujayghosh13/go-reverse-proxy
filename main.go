package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Backend represents a backend server
type Backend struct {
	Address           string
	Healthy           bool
	ActiveConnections int
}

// List of backend servers
var backends []*Backend

// Variables for backend locking and Graceful Shutdown
var (
	counter        int
	mu             sync.Mutex
	wg             sync.WaitGroup
	isShuttingDown bool
	shutdownMu     sync.Mutex
)

const (
	maxIdleConnsPerBackend = 10
	maxIdleTime            = 30 * time.Second
)

// pooledConn wraps a net.Conn with its return timestamp
type pooledConn struct {
	conn       net.Conn
	returnedAt time.Time
}

// ConnPool manages reusable TCP connections
type ConnPool struct {
	mu    sync.Mutex
	conns map[string][]pooledConn
}

// Create a new connection pool
func NewConnPool() *ConnPool {
	return &ConnPool{
		conns: make(map[string][]pooledConn),
	}
}

// Get a connection from the pool
func (p *ConnPool) Get(address string) (net.Conn, error) {
	p.mu.Lock()
	for len(p.conns[address]) > 0 {
		conns := p.conns[address]
		pc := conns[len(conns)-1]
		p.conns[address] = conns[:len(conns)-1]

		if time.Since(pc.returnedAt) > maxIdleTime {
			pc.conn.Close()
			fmt.Println("Discarded stale pooled connection to", address)
			continue
		}

		p.mu.Unlock()
		fmt.Println("Reused a pooled connection to", address)
		return pc.conn, nil
	}
	p.mu.Unlock()

	fmt.Println("No pooled connection available, dialing fresh to", address)
	return net.Dial("tcp", address)
}

// Put a connection back into the pool
func (p *ConnPool) Put(address string, conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns[address]) >= maxIdleConnsPerBackend {
		conn.Close()
		fmt.Println("Pool full for", address, "- closed connection")
		return
	}

	p.conns[address] = append(p.conns[address], pooledConn{
		conn:       conn,
		returnedAt: time.Now(),
	})

	fmt.Println("Returned connection to pool for", address)
}

// Create the global connection pool
var pool = NewConnPool()

// Select the healthy backend with the fewest active connections
func getLeastConnectionsBackend() *Backend {
	mu.Lock()
	defer mu.Unlock()

	var best *Backend
	for _, backend := range backends {
		if !backend.Healthy {
			continue
		}
		if best == nil || backend.ActiveConnections < best.ActiveConnections {
			best = backend
		}
	}

	return best
}

// Check backend health periodically
func healthCheck(intervalSeconds int) {
	interval := time.Duration(intervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for {
		for _, b := range backends {

			resp, err := http.Get("http://" + b.Address)

			if err != nil {

				// Backend has gone down
				if b.Healthy {
					fmt.Println("Backend went DOWN:", b.Address)
				}

				b.Healthy = false

			} else {

				resp.Body.Close()

				// Backend came back online
				if !b.Healthy {
					fmt.Println("Backend came back UP:", b.Address)
				}

				b.Healthy = true
			}
		}

		time.Sleep(interval)
	}
}

// Read the HTTP request from the client
func readRequest(clientConn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(clientConn)

	var request []byte
	contentLength := 0

	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			return nil, err
		}

		request = append(request, []byte(line)...)

		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if cl, err := strconv.Atoi(val); err == nil {
					contentLength = cl
				}
			}
		}

		// Empty line means HTTP headers are finished
		if line == "\r\n" {
			break
		}
	}

	if contentLength > 0 {
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return nil, err
		}
		request = append(request, body...)
	}

	return request, nil
}

// Forward the client request to the selected backend
func forwardToBackend(
	target *Backend,
	request []byte,
) (*http.Response, net.Conn, error) {

	// Get a connection from the pool
	backendConn, err := pool.Get(target.Address)

	if err != nil {
		return nil, nil, err
	}

	// Send request to backend
	_, writeErr := backendConn.Write(request)
	var resp *http.Response

	if writeErr == nil {
		reader := bufio.NewReader(backendConn)
		resp, err = http.ReadResponse(reader, nil)
	}

	// If writing or reading failed on a pooled connection, retry with a fresh connection!
	if writeErr != nil || err != nil {

		// The pooled connection may be stale or closed
		fmt.Println("Pooled connection failed or stale, creating a fresh connection")

		backendConn.Close()

		// Create a fresh connection
		backendConn, err = net.Dial("tcp", target.Address)

		if err != nil {
			return nil, nil, err
		}

		// Try sending the request again
		_, writeErr = backendConn.Write(request)

		if writeErr != nil {
			backendConn.Close()

			return nil, nil, writeErr
		}

		reader := bufio.NewReader(backendConn)

		resp, err = http.ReadResponse(reader, nil)

		if err != nil {
			backendConn.Close()

			return nil, nil, err
		}
	}

	return resp, backendConn, nil
}

// Handle each client connection
func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Extract client IP (host only, stripping ephemeral port)
	clientIP, _, err := net.SplitHostPort(clientConn.RemoteAddr().String())
	if err != nil {
		clientIP = clientConn.RemoteAddr().String()
	}

	// Read client's HTTP request to drain incoming TCP receive buffer
	request, err := readRequest(clientConn)
	if err != nil {
		fmt.Println("Error reading client request:", err)
		return
	}

	// Rate limit check per client IP
	if !globalRateLimiter.Allow(clientIP) {
		globalMetrics.RecordRequest("", true)
		clientConn.Write([]byte(
			"HTTP/1.1 429 Too Many Requests\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"Content-Length: 19\r\n" +
				"Connection: close\r\n\r\n" +
				"Rate limit exceeded",
		))
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		return
	}

	// Select the healthy backend with the fewest active connections
	target := getLeastConnectionsBackend()

	if target == nil {
		globalMetrics.RecordRequest("", true)

		// All backends are unavailable
		clientConn.Write([]byte(
			"HTTP/1.1 503 Service Unavailable\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"Content-Length: 21\r\n" +
				"Connection: close\r\n\r\n" +
				"All backends are down",
		))
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}

		return
	}

	// Increment active connections for selected backend
	mu.Lock()
	target.ActiveConnections++
	activeCount := target.ActiveConnections
	mu.Unlock()

	// Ensure active connections count is decremented when handleConnection exits
	defer func() {
		mu.Lock()
		target.ActiveConnections--
		mu.Unlock()
	}()

	fmt.Println("Least-connections selected:", target.Address, "active:", activeCount)

	// Forward request to backend
	resp, backendConn, err := forwardToBackend(target, request)

	if err != nil {
		globalMetrics.RecordRequest(target.Address, true)
		fmt.Println("Error talking to backend:", err)

		clientConn.Write([]byte(
			"HTTP/1.1 502 Bad Gateway\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"Content-Length: 13\r\n" +
				"Connection: close\r\n\r\n" +
				"Backend error",
		))
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}

		return
	}

	// Send backend response to client
	err = resp.Write(clientConn)

	if err != nil {
		globalMetrics.RecordRequest(target.Address, true)
		fmt.Println("Error writing response to client:", err)

		backendConn.Close()

		return
	}

	globalMetrics.RecordRequest(target.Address, false)

	// Put the backend connection back into the pool
	pool.Put(target.Address, backendConn)
}

func main() {

	// Load configuration from config.yaml
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Println("Error loading config file:", err)
		return
	}

	// Initialize backend list from config
	for _, addr := range cfg.Backends {
		backends = append(backends, &Backend{
			Address: addr,
			Healthy: true,
		})
	}

	// Start HTTP metrics server on port 9090
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", globalMetrics.MetricsHandler)
		fmt.Println("Metrics server listening on port 9090...")
		if err := http.ListenAndServe(":9090", mux); err != nil {
			fmt.Println("Metrics server error:", err)
		}
	}()

	// Start reverse proxy on configured port
	listenAddr := fmt.Sprintf(":%d", cfg.Port)
	listener, err := net.Listen("tcp", listenAddr)

	if err != nil {

		fmt.Println("Error starting server:", err)

		return
	}

	// Run health checking in a separate goroutine
	go healthCheck(cfg.HealthCheckIntervalSeconds)

	fmt.Printf("Proxy is listening on port %d...\n", cfg.Port)

	// Listen for shutdown signals (Ctrl+C / SIGTERM)
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		fmt.Println("Shutdown signal received. Finishing in-flight requests...")

		shutdownMu.Lock()
		isShuttingDown = true
		shutdownMu.Unlock()

		listener.Close()
	}()

	// Accept client connections
	for {

		conn, err := listener.Accept()

		if err != nil {
			shutdownMu.Lock()
			shuttingDown := isShuttingDown
			shutdownMu.Unlock()

			if shuttingDown {
				break
			}

			fmt.Println("Error accepting connection:", err)

			continue
		}

		shutdownMu.Lock()
		if isShuttingDown {
			shutdownMu.Unlock()
			conn.Close()
			break
		}

		wg.Add(1)
		shutdownMu.Unlock()

		// Handle every client concurrently and track in-flight requests
		go func(c net.Conn) {
			defer wg.Done()
			handleConnection(c)
		}(conn)
	}

	// Wait for all in-flight requests to complete
	wg.Wait()
	fmt.Println("All requests completed. Shutting down cleanly.")
}
