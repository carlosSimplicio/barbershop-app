package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment     string
	HTTPAddress     string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	config := Config{
		Environment:     valueOrDefault(lookup, "APP_ENV", "development"),
		HTTPAddress:     valueOrDefault(lookup, "HTTP_ADDR", ":8080"),
		DatabaseURL:     valueOrDefault(lookup, "DATABASE_URL", ""),
		LogLevel:        valueOrDefault(lookup, "LOG_LEVEL", "info"),
		ShutdownTimeout: 10 * time.Second,
	}

	if config.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if config.HTTPAddress == "" {
		return Config{}, errors.New("HTTP_ADDR is required")
	}

	switch config.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}

	shutdownTimeout := valueOrDefault(lookup, "SHUTDOWN_TIMEOUT", "10s")
	parsedTimeout, err := time.ParseDuration(shutdownTimeout)
	if err != nil || parsedTimeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration")
	}
	config.ShutdownTimeout = parsedTimeout

	switch config.Environment {
	case "development", "test", "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}

	return config, nil
}

func valueOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok {
		return value
	}
	return fallback
}
