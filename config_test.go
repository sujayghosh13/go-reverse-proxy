package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yamlData := []byte(`
port: 8080
health_check_interval_seconds: 5
circuit_breaker_threshold: 4
circuit_breaker_cooldown_seconds: 10
log_level: "DEBUG"
log_json: true
backends:
  - localhost:9001
  - localhost:9002
`)

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(yamlData); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpFile.Close()

	cfg, err := loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.HealthCheckIntervalSeconds != 5 {
		t.Errorf("expected health check 5, got %d", cfg.HealthCheckIntervalSeconds)
	}
	if cfg.CircuitBreakerThreshold != 4 {
		t.Errorf("expected cb threshold 4, got %d", cfg.CircuitBreakerThreshold)
	}
	if len(cfg.Backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(cfg.Backends))
	}
}
