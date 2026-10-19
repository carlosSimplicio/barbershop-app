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

func TestLoadConfiguresDevelopmentSMTP(t *testing.T) {
	values := validValues(t)
	config, err := load(mapLookup(values))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.EmailProvider != "smtp" || config.SMTPHost != "mailpit" || config.SMTPPort != 1025 {
		t.Fatalf("unexpected SMTP config: %#v", config)
	}
	if config.PublicAppURL != "https://localhost:8443" || config.SMTPFrom != "no-reply@barbershop.local" {
		t.Fatalf("unexpected email identity config: %#v", config)
	}
}

func TestLoadAllowsDisabledEmailOnlyInTests(t *testing.T) {
	values := validValues(t)
	values["APP_ENV"] = "test"
	values["EMAIL_PROVIDER"] = "disabled"
	delete(values, "PUBLIC_APP_URL")
	delete(values, "SMTP_HOST")
	delete(values, "SMTP_PORT")
	delete(values, "SMTP_FROM")
	if _, err := load(mapLookup(values)); err != nil {
		t.Fatalf("test config should allow disabled email: %v", err)
	}

	values["APP_ENV"] = "development"
	values["PUBLIC_APP_URL"] = "https://localhost:8443"
	values["SMTP_HOST"] = "mailpit"
	values["SMTP_PORT"] = "1025"
	values["SMTP_FROM"] = "no-reply@barbershop.local"
	if _, err := load(mapLookup(values)); err == nil {
		t.Fatal("development config allowed disabled transactional email")
	}
}

func TestLoadRejectsLocalSMTPInProductionWithoutRealProvider(t *testing.T) {
	values := validValues(t)
	values["APP_ENV"] = "production"
	if _, err := load(mapLookup(values)); err == nil {
		t.Fatal("production config allowed development SMTP provider")
	}
}

func TestLoadRejectsInvalidEmailConfiguration(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{key: "PUBLIC_APP_URL", value: "http://example.com"},
		{key: "PUBLIC_APP_URL", value: "https://example.com/path"},
		{key: "SMTP_HOST", value: ""},
		{key: "SMTP_PORT", value: "70000"},
		{key: "SMTP_FROM", value: "invalid"},
	}
	for _, test := range tests {
		t.Run(test.key+"="+test.value, func(t *testing.T) {
			values := validValues(t)
			values[test.key] = test.value
			if _, err := load(mapLookup(values)); err == nil {
				t.Fatal("invalid email configuration was accepted")
			}
		})
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
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
		"TRUSTED_ORIGINS":            "https://localhost:8443",
		"TRUSTED_PROXY_CIDRS":        "172.30.0.4/32",
		"EMAIL_PROVIDER":             "smtp",
		"PUBLIC_APP_URL":             "https://localhost:8443",
		"SMTP_HOST":                  "mailpit",
		"SMTP_PORT":                  "1025",
		"SMTP_FROM":                  "no-reply@barbershop.local",
	}
}
