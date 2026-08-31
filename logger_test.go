package main

import (
	"errors"
	"log/slog"
	"testing"
	"time"
)

func TestInitLogger(t *testing.T) {
	InitLogger("DEBUG", true)
	if Logger == nil {
		t.Fatalf("expected Logger to be initialized")
	}

	InitLogger("WARN", false)
	if Logger == nil {
		t.Fatalf("expected Logger to be initialized")
	}
}

func TestLogRequest(t *testing.T) {
	InitLogger("DEBUG", true)
	LogRequest(slog.LevelInfo, "Request processed", "req-12345", "GET", "localhost:9001", 200, 15*time.Millisecond, nil)
	LogRequest(slog.LevelError, "Backend failed", "req-67890", "POST", "localhost:9002", 502, 50*time.Millisecond, errors.New("connection refused"))
}
