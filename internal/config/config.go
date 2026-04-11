package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all service configuration loaded from environment variables.
type Config struct {
	Server  ServerConfig
	Kafka   KafkaConfig
	Metrics MetricsConfig
	Log     LogConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// KafkaConfig holds Kafka producer settings.
type KafkaConfig struct {
	Brokers      []string
	SSPTopic     string
	DSPTopic     string
	BatchSize    int
	WriteTimeout time.Duration
	RequiredAcks int
}

// MetricsConfig holds Prometheus metrics server settings.
type MetricsConfig struct {
	Port int
	Path string
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string
	Format string // json or console
}

// Load reads configuration from environment variables, applying defaults where appropriate.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Kafka: KafkaConfig{
			Brokers:      getEnvStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			SSPTopic:     getEnvString("KAFKA_SSP_TOPIC", "ad-events-ssp"),
			DSPTopic:     getEnvString("KAFKA_DSP_TOPIC", "ad-events-dsp"),
			BatchSize:    getEnvInt("KAFKA_BATCH_SIZE", 100),
			WriteTimeout: getEnvDuration("KAFKA_WRITE_TIMEOUT", 10*time.Second),
			RequiredAcks: getEnvInt("KAFKA_REQUIRED_ACKS", 1),
		},
		Metrics: MetricsConfig{
			Port: getEnvInt("METRICS_PORT", 9090),
			Path: getEnvString("METRICS_PATH", "/metrics"),
		},
		Log: LogConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", c.Server.Port)
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS must not be empty")
	}
	if c.Kafka.SSPTopic == "" {
		return fmt.Errorf("KAFKA_SSP_TOPIC must not be empty")
	}
	if c.Kafka.DSPTopic == "" {
		return fmt.Errorf("KAFKA_DSP_TOPIC must not be empty")
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Log.Level)] {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}
	return nil
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}

func getEnvStringSlice(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}
