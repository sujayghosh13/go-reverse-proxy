package main

import (
	"net"
	"testing"
	"time"
)

func TestConnPool_GetAndPut(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := l.Addr().String()
	pool := NewConnPool()

	// 1. Get from empty pool -> dials fresh connection
	conn1, err := pool.Get(addr)
	if err != nil {
		t.Fatalf("failed to get connection: %v", err)
	}

	// 2. Put connection back
	pool.Put(addr, conn1)

	// 3. Get connection back -> reuses pooled connection
	conn2, err := pool.Get(addr)
	if err != nil {
		t.Fatalf("failed to get connection from pool: %v", err)
	}
	conn2.Close()
}

func TestConnPool_MaxPoolLimits(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := l.Addr().String()
	pool := NewConnPool()

	// Put max + 2 connections
	for i := 0; i < maxIdleConnsPerBackend+2; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		pool.Put(addr, conn)
	}

	pool.mu.Lock()
	count := len(pool.conns[addr])
	pool.mu.Unlock()

	if count > maxIdleConnsPerBackend {
		t.Fatalf("expected max %d pooled conns, got %d", maxIdleConnsPerBackend, count)
	}
}

func TestConnPool_StaleExpiration(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := l.Addr().String()
	pool := NewConnPool()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	// Manually inject a stale connection (> 30s old)
	pool.mu.Lock()
	pool.conns[addr] = []pooledConn{
		{
			conn:       conn,
			returnedAt: time.Now().Add(-35 * time.Second),
		},
	}
	pool.mu.Unlock()

	// Get should discard stale connection and dial fresh
	connFresh, err := pool.Get(addr)
	if err != nil {
		t.Fatalf("expected fresh dial after stale discard, got error: %v", err)
	}
	connFresh.Close()
}
