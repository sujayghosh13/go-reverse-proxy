package main

import (
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"
)

// RecoverConnection handles panics per connection goroutine cleanly
func RecoverConnection(clientConn net.Conn, requestID string) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		LogRequest(slog.LevelError, "Panic recovered in connection handler", requestID, "", "", 500, 0, fmt.Errorf("panic: %v", r))
		if Logger != nil {
			Logger.Error("Panic stacktrace", "request_id", requestID, "stack", stack)
		}

		if clientConn != nil {
			clientConn.Write([]byte(
				"HTTP/1.1 500 Internal Server Error\r\n" +
					"Content-Type: text/plain; charset=utf-8\r\n" +
					"Content-Length: 21\r\n" +
					"Connection: close\r\n\r\n" +
					"Internal Server Error",
			))
			if tcpConn, ok := clientConn.(*net.TCPConn); ok {
				tcpConn.CloseWrite()
			}
		}
	}
}

// SetConnTimeouts applies read and write deadlines to a net.Conn
func SetConnTimeouts(conn net.Conn, readTimeout time.Duration, writeTimeout time.Duration) {
	if conn == nil {
		return
	}
	now := time.Now()
	if readTimeout > 0 {
		conn.SetReadDeadline(now.Add(readTimeout))
	}
	if writeTimeout > 0 {
		conn.SetWriteDeadline(now.Add(writeTimeout))
	}
}
