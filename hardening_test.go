package main

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestRecoverConnection_RecoversPanic(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	addr := l.Addr().String()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer RecoverConnection(conn, "req-panic-test")
		panic("simulated connection panic")
	}()

	clientConn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer clientConn.Close()

	buf := make([]byte, 512)
	clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _ := clientConn.Read(buf)

	respStr := string(buf[:n])
	if !strings.Contains(respStr, "500 Internal Server Error") {
		t.Fatalf("expected 500 response on panic recovery, got:\n%s", respStr)
	}
}

func TestSetConnTimeouts(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	SetConnTimeouts(conn, 100*time.Millisecond, 100*time.Millisecond)
}
