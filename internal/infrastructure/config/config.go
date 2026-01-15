package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Redis      RedisConfig      `yaml:"redis"`
	Worker     WorkerConfig     `yaml:"worker"`
	Simulation SimulationConfig `yaml:"simulation"`
	AI         AIConfig         `yaml:"ai"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr          string `yaml:"addr"`            // For local Redis: "localhost:6379"
	URL           string `yaml:"url"`             // For cloud Redis (Upstash): "rediss://default:password@endpoint:port"
	Password      string `yaml:"password"`        // Optional password for simple auth
	DB            int    `yaml:"db"`              // Database number (default 0)
	TLSSkipVerify bool   `yaml:"tls_skip_verify"` // Skip TLS certificate verification (for Upstash in Docker)
}

// WorkerConfig represents worker configuration
type WorkerConfig struct {
	MaxAttempts   int `yaml:"max_attempts"`
	BaseBackoffMs int `yaml:"base_backoff_ms"`
}

// SimulationConfig represents failure simulation configuration
type SimulationConfig struct {
	Enabled     bool    `yaml:"enabled"`
	FailureRate float64 `yaml:"failure_rate"`
}

// AIConfig represents AI service configuration
type AIConfig struct {
	OllamaURL   string `yaml:"ollama_url"`
	InsightsURL string `yaml:"insights_url"` // URL for remote insights service (optional)
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Check for CONFIG_ENV environment variable to determine config file
	configEnv := os.Getenv("CONFIG_ENV")
	if configEnv == "" {
		configEnv = "dev" // Default to dev if not specified
	}

	// Use environment-specific config
	path = fmt.Sprintf("configs/config.%s.yaml", configEnv)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
