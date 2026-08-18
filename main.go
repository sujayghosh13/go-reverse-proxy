package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type Backend struct {
	Address string
	Healthy bool
}

var backends = []*Backend{
	{Address: "localhost:9001", Healthy: true},
	{Address: "localhost:9002", Healthy: true},
	{Address: "localhost:9003", Healthy: true},
}

var (
	counter int
	mu      sync.Mutex
)

func getNextBackend() *Backend {
	mu.Lock()
	defer mu.Unlock()

	for i := 0; i < len(backends); i++ {
		backend := backends[counter%len(backends)]
		counter++
		if backend.Healthy {
			return backend
		}
	}
	return nil // no healthy backends
}

func healthCheck() {
	for {
		for _, b := range backends {
			resp, err := http.Get("http://" + b.Address)
			if err != nil {
				if b.Healthy {
					fmt.Println("Backend went DOWN:", b.Address)
				}
				b.Healthy = false
			} else {
				resp.Body.Close()
				if !b.Healthy {
					fmt.Println("Backend came back UP:", b.Address)
				}
				b.Healthy = true
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	go healthCheck()

	fmt.Println("Proxy is listening on port 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	target := getNextBackend()
	if target == nil {
		clientConn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\nAll backends are down"))
		return
	}
	fmt.Println("Forwarding to:", target.Address)

	backendConn, err := net.Dial("tcp", target.Address)
	if err != nil {
		fmt.Println("Error connecting to backend:", err)
		return
	}
	defer backendConn.Close()

	reader := bufio.NewReader(clientConn)
	var request []byte
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		request = append(request, []byte(line)...)
		if line == "\r\n" {
			break
		}
	}

	backendConn.Write(request)
	if tcpConn, ok := backendConn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}

	io.Copy(clientConn, backendConn)
}