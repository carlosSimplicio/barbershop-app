package config

import (
	"testing"
	"time"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := load(func(string) (string, bool) { return "", false })

	if err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadAppliesSafeDefaults(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://example"}
	config, err := load(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Environment != "development" {
		t.Errorf("Environment = %q, want development", config.Environment)
	}
	if config.HTTPAddress != ":8080" {
		t.Errorf("HTTPAddress = %q, want :8080", config.HTTPAddress)
	}
	if config.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 10s", config.ShutdownTimeout)
	}
}

func TestLoadRejectsUnknownLogLevel(t *testing.T) {
	values := map[string]string{
		"DATABASE_URL": "postgres://example",
		"LOG_LEVEL":    "verbose",
	}
	_, err := load(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})

	if err == nil {
		t.Fatal("expected unknown LOG_LEVEL to fail")
	}
}
