package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkProxy_ConnectionPooling(b *testing.B) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		fmt.Fprint(w, "OK")
	}))
	defer backend.Close()

	addr := backend.Listener.Addr().String()
	pool := NewConnPool()

	rawReq := []byte("GET /benchmark HTTP/1.1\r\nHost: localhost\r\n\r\n")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, err := pool.Get(addr)
		if err != nil {
			b.Fatalf("pool Get failed: %v", err)
		}

		if _, err := conn.Write(rawReq); err != nil {
			conn.Close()
			b.Fatalf("conn write failed: %v", err)
		}

		buf := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			conn.Close()
			b.Fatalf("conn read failed: %v", err)
		}
		_ = n

		pool.Put(addr, conn)
	}
}

func BenchmarkProxy_DirectDialingNoPooling(b *testing.B) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		fmt.Fprint(w, "OK")
	}))
	defer backend.Close()

	addr := backend.Listener.Addr().String()
	rawReq := []byte("GET /benchmark HTTP/1.1\r\nHost: localhost\r\n\r\n")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatalf("dial failed: %v", err)
		}

		if _, err := conn.Write(rawReq); err != nil {
			conn.Close()
			b.Fatalf("conn write failed: %v", err)
		}

		buf := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			conn.Close()
			b.Fatalf("conn read failed: %v", err)
		}
		_ = n

		conn.Close()
	}
}
