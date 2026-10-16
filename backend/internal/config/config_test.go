package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
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
	values := validValues(t)
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
	values := validValues(t)
	values["LOG_LEVEL"] = "verbose"
	_, err := load(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})

	if err == nil {
		t.Fatal("expected unknown LOG_LEVEL to fail")
	}
}

func TestLoadRejectsMismatchedJWTKeys(t *testing.T) {
	values := validValues(t)
	otherSeed := make([]byte, ed25519.SeedSize)
	for index := range otherSeed {
		otherSeed[index] = 9
	}
	values["JWT_PRIVATE_KEY_BASE64"] = base64.RawStdEncoding.EncodeToString(otherSeed)

	_, err := load(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected mismatched JWT keys to fail")
	}
}

func validValues(t *testing.T) map[string]string {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKeys, err := json.Marshal(map[string]string{
		"test-key": base64.RawStdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)),
	})
	if err != nil {
		t.Fatalf("marshal public keys: %v", err)
	}
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index + 11)
	}
	encodedKey := base64.RawStdEncoding.EncodeToString(key)
	return map[string]string{
		"DATABASE_URL":               "postgres://example",
		"JWT_ACTIVE_KEY_ID":          "test-key",
		"JWT_PRIVATE_KEY_BASE64":     base64.RawStdEncoding.EncodeToString(seed),
		"JWT_PUBLIC_KEYS_JSON":       string(publicKeys),
		"CSRF_SIGNING_KEY_BASE64":    encodedKey,
		"RATE_LIMIT_HMAC_KEY_BASE64": encodedKey,
	}
}
