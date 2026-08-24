package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Backend represents a backend server
type Backend struct {
	Address string
	Healthy bool
}

// List of backend servers
var backends = []*Backend{
	{Address: "localhost:9001", Healthy: true},
	{Address: "localhost:9002", Healthy: true},
	{Address: "localhost:9003", Healthy: true},
}

// Variables for Round Robin
var (
	counter int
	mu      sync.Mutex
)

// ConnPool manages reusable TCP connections
type ConnPool struct {
	mu    sync.Mutex
	conns map[string][]net.Conn
}

// Create a new connection pool
func NewConnPool() *ConnPool {
	return &ConnPool{
		conns: make(map[string][]net.Conn),
	}
}

// Get a connection from the pool
func (p *ConnPool) Get(address string) (net.Conn, error) {
	p.mu.Lock()
	conns := p.conns[address]
	if len(conns) > 0 {
		conn := conns[len(conns)-1]
		p.conns[address] = conns[:len(conns)-1]
		p.mu.Unlock()
		fmt.Println("Reused a pooled connection to", address)
		return conn, nil
	}
	p.mu.Unlock()
	fmt.Println("No pooled connection available, dialing fresh to", address)
	return net.Dial("tcp", address)
}

// Put a connection back into the pool
func (p *ConnPool) Put(address string, conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.conns[address] = append(p.conns[address], conn)

	fmt.Println("Returned connection to pool for", address)
}

// Create the global connection pool
var pool = NewConnPool()

// Select the next healthy backend using Round Robin
func getNextBackend() *Backend {
	mu.Lock()
	defer mu.Unlock()

	for i := 0; i < len(backends); i++ {
		backend := backends[counter%len(backends)]

		counter++

		// Only select healthy backends
		if backend.Healthy {
			return backend
		}
	}

	// All backends are down
	return nil
}

// Check backend health every 5 seconds
func healthCheck() {
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

		time.Sleep(5 * time.Second)
	}
}

// Read the HTTP request from the client
func readRequest(clientConn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(clientConn)

	var request []byte

	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			return nil, err
		}

		request = append(request, []byte(line)...)

		// Empty line means HTTP headers are finished
		if line == "\r\n" {
			break
		}
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

	if writeErr != nil {

		// The pooled connection may be stale or closed
		fmt.Println("Pooled connection failed, creating a fresh connection")

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
	}

	// Read backend response
	reader := bufio.NewReader(backendConn)

	resp, err := http.ReadResponse(reader, nil)

	if err != nil {
		backendConn.Close()

		return nil, nil, err
	}

	return resp, backendConn, nil
}

// Handle each client connection
func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Select the next healthy backend
	target := getNextBackend()

	if target == nil {

		// All backends are unavailable
		clientConn.Write([]byte(
			"HTTP/1.1 503 Service Unavailable\r\n\r\nAll backends are down",
		))

		return
	}

	fmt.Println("Selected backend:", target.Address)

	// Read the client's HTTP request
	request, err := readRequest(clientConn)

	if err != nil {
		fmt.Println("Error reading client request:", err)

		return
	}

	// Forward request to backend
	resp, backendConn, err := forwardToBackend(target, request)

	if err != nil {

		fmt.Println("Error talking to backend:", err)

		clientConn.Write([]byte(
			"HTTP/1.1 502 Bad Gateway\r\n\r\nBackend error",
		))

		return
	}

	// Send backend response to client
	err = resp.Write(clientConn)

	if err != nil {

		fmt.Println("Error writing response to client:", err)

		backendConn.Close()

		return
	}

	// Put the backend connection back into the pool
	pool.Put(target.Address, backendConn)
}

func main() {

	// Start reverse proxy on port 8080
	listener, err := net.Listen("tcp", ":8080")

	if err != nil {

		fmt.Println("Error starting server:", err)

		return
	}

	defer listener.Close()

	// Run health checking in a separate goroutine
	go healthCheck()

	fmt.Println("Proxy is listening on port 8080...")

	// Accept client connections
	for {

		conn, err := listener.Accept()

		if err != nil {

			fmt.Println("Error accepting connection:", err)

			continue
		}

		// Handle every client concurrently
		go handleConnection(conn)
	}
}
