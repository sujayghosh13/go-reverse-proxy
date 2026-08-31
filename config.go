package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the proxy configuration
type Config struct {
	Port                          int      `yaml:"port"`
	HealthCheckIntervalSeconds    int      `yaml:"health_check_interval_seconds"`
	Backends                      []string `yaml:"backends"`
	CircuitBreakerThreshold       int      `yaml:"circuit_breaker_threshold"`
	CircuitBreakerCooldownSeconds int      `yaml:"circuit_breaker_cooldown_seconds"`
	LogLevel                      string   `yaml:"log_level"`
	LogJSON                       bool     `yaml:"log_json"`
}

// loadConfig reads and unmarshals the YAML configuration file from path
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
