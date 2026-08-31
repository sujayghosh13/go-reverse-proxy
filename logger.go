package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"
)

var Logger *slog.Logger

// InitLogger initializes the global structured logger with level and format
func InitLogger(levelStr string, isJSON bool) {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if isJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

func init() {
	InitLogger("INFO", true)
}

// LogRequest logs an HTTP request with structured attributes
func LogRequest(level slog.Level, msg string, requestID string, method string, backend string, status int, latency time.Duration, err error) {
	if Logger == nil {
		InitLogger("INFO", true)
	}

	attrs := []slog.Attr{}
	if requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}
	if method != "" {
		attrs = append(attrs, slog.String("method", method))
	}
	if backend != "" {
		attrs = append(attrs, slog.String("backend", backend))
	}
	if status > 0 {
		attrs = append(attrs, slog.Int("status", status))
	}
	if latency > 0 {
		attrs = append(attrs, slog.Int64("latency_ms", latency.Milliseconds()))
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	Logger.LogAttrs(context.Background(), level, msg, attrs...)
}
