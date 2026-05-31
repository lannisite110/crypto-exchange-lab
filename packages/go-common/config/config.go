package config

import (
	"os"
	"strconv"
	"time"
)

// Base holds shared service configuration loaded from environment variables.
type Base struct {
	ServiceName string
	Environment string
	HTTPPort    int
	LogLevel    string

	PostgresURL string
	RedisURL    string
}

// Load reads configuration from environment with sensible local defaults.
func Load(serviceName string) Base {
	return Base{
		ServiceName: serviceName,
		Environment: getEnv("APP_ENV", "local"),
		HTTPPort:    getEnvInt("HTTP_PORT", 8080),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		PostgresURL: getEnv("DATABASE_URL", "postgres://lab:lab@localhost:5432/crypto_exchange_lab?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// HTTPReadTimeout returns the default HTTP server read timeout.
func HTTPReadTimeout() time.Duration {
	return 10 * time.Second
}

// HTTPWriteTimeout returns the default HTTP server write timeout.
func HTTPWriteTimeout() time.Duration {
	return 10 * time.Second
}
