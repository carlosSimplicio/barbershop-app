package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment       string
	HTTPAddress       string
	DatabaseURL       string
	LogLevel          string
	ShutdownTimeout   time.Duration
	Argon2MemoryKiB   uint32
	Argon2Iterations  uint32
	Argon2Parallelism uint8
	JWTActiveKeyID    string
	JWTPrivateKey     ed25519.PrivateKey
	JWTPublicKeys     map[string]ed25519.PublicKey
	CSRFSigningKey    []byte
	RateLimitHMACKey  []byte
	TrustedOrigins    []string
	TrustedProxyCIDRs []string
	EmailProvider     string
	PublicAppURL      string
	SMTPHost          string
	SMTPPort          uint16
	SMTPFrom          string
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	config := Config{
		Environment:       valueOrDefault(lookup, "APP_ENV", "development"),
		HTTPAddress:       valueOrDefault(lookup, "HTTP_ADDR", ":8080"),
		DatabaseURL:       valueOrDefault(lookup, "DATABASE_URL", ""),
		LogLevel:          valueOrDefault(lookup, "LOG_LEVEL", "info"),
		ShutdownTimeout:   10 * time.Second,
		Argon2MemoryKiB:   19 * 1024,
		Argon2Iterations:  2,
		Argon2Parallelism: 1,
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

	if err := loadAuthenticationConfig(&config, lookup); err != nil {
		return Config{}, err
	}
	if err := loadEmailConfig(&config, lookup); err != nil {
		return Config{}, err
	}

	return config, nil
}

func loadEmailConfig(config *Config, lookup func(string) (string, bool)) error {
	config.EmailProvider = valueOrDefault(lookup, "EMAIL_PROVIDER", "smtp")
	config.PublicAppURL = valueOrDefault(lookup, "PUBLIC_APP_URL", "")
	config.SMTPHost = valueOrDefault(lookup, "SMTP_HOST", "")
	config.SMTPFrom = valueOrDefault(lookup, "SMTP_FROM", "")
	if config.EmailProvider != "smtp" && config.EmailProvider != "disabled" {
		return errors.New("EMAIL_PROVIDER must be smtp or disabled")
	}
	if config.Environment == "production" {
		return errors.New("production transactional email requires a supported real provider")
	}
	if config.EmailProvider == "disabled" && config.Environment != "test" {
		return errors.New("EMAIL_PROVIDER=disabled is allowed only in tests")
	}
	if config.EmailProvider == "disabled" {
		return nil
	}

	publicURL, err := url.Parse(config.PublicAppURL)
	if err != nil || publicURL.Scheme != "https" || publicURL.Host == "" || publicURL.Path != "" || publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return errors.New("PUBLIC_APP_URL must be an HTTPS origin without path")
	}
	port, err := uint32Value(lookup, "SMTP_PORT", 0)
	if err != nil || port == 0 || port > 65535 {
		return errors.New("SMTP_PORT must be between 1 and 65535")
	}
	config.SMTPPort = uint16(port)
	if config.EmailProvider == "smtp" {
		if config.SMTPHost == "" || strings.ContainsAny(config.SMTPHost, " /:@") {
			return errors.New("SMTP_HOST must be a host name without a port")
		}
		from, parseErr := mail.ParseAddress(config.SMTPFrom)
		if parseErr != nil || !strings.Contains(from.Address, "@") || strings.ContainsAny(config.SMTPFrom, "\r\n") {
			return errors.New("SMTP_FROM must be a valid email address")
		}
	}
	return nil
}

func loadAuthenticationConfig(config *Config, lookup func(string) (string, bool)) error {
	var err error
	config.Argon2MemoryKiB, err = uint32Value(lookup, "ARGON2_MEMORY_KIB", config.Argon2MemoryKiB)
	if err != nil || config.Argon2MemoryKiB < 19*1024 || config.Argon2MemoryKiB > 1024*1024 {
		return errors.New("ARGON2_MEMORY_KIB must be between 19456 and 1048576")
	}
	config.Argon2Iterations, err = uint32Value(lookup, "ARGON2_ITERATIONS", config.Argon2Iterations)
	if err != nil || config.Argon2Iterations < 2 || config.Argon2Iterations > 10 {
		return errors.New("ARGON2_ITERATIONS must be between 2 and 10")
	}
	parallelism, err := uint32Value(lookup, "ARGON2_PARALLELISM", uint32(config.Argon2Parallelism))
	if err != nil || parallelism < 1 || parallelism > 16 {
		return errors.New("ARGON2_PARALLELISM must be between 1 and 16")
	}
	config.Argon2Parallelism = uint8(parallelism)

	config.JWTActiveKeyID = valueOrDefault(lookup, "JWT_ACTIVE_KEY_ID", "")
	if config.JWTActiveKeyID == "" {
		return errors.New("JWT_ACTIVE_KEY_ID is required")
	}
	privateSeed, err := requiredBase64(lookup, "JWT_PRIVATE_KEY_BASE64", ed25519.SeedSize)
	if err != nil {
		return err
	}
	config.JWTPrivateKey = ed25519.NewKeyFromSeed(privateSeed)

	publicKeysJSON := valueOrDefault(lookup, "JWT_PUBLIC_KEYS_JSON", "")
	var encodedPublicKeys map[string]string
	if err := json.Unmarshal([]byte(publicKeysJSON), &encodedPublicKeys); err != nil || len(encodedPublicKeys) == 0 {
		return errors.New("JWT_PUBLIC_KEYS_JSON must be a non-empty JSON object")
	}
	config.JWTPublicKeys = make(map[string]ed25519.PublicKey, len(encodedPublicKeys))
	for keyID, encoded := range encodedPublicKeys {
		decoded, err := base64.RawStdEncoding.Strict().DecodeString(encoded)
		if err != nil || len(decoded) != ed25519.PublicKeySize || keyID == "" {
			return errors.New("JWT_PUBLIC_KEYS_JSON contains an invalid Ed25519 public key")
		}
		config.JWTPublicKeys[keyID] = ed25519.PublicKey(decoded)
	}
	activePublicKey, ok := config.JWTPublicKeys[config.JWTActiveKeyID]
	if !ok || !config.JWTPrivateKey.Public().(ed25519.PublicKey).Equal(activePublicKey) {
		return errors.New("active JWT public key does not match JWT_PRIVATE_KEY_BASE64")
	}

	config.CSRFSigningKey, err = requiredBase64AtLeast(lookup, "CSRF_SIGNING_KEY_BASE64", 32)
	if err != nil {
		return err
	}
	config.RateLimitHMACKey, err = requiredBase64AtLeast(lookup, "RATE_LIMIT_HMAC_KEY_BASE64", 32)
	if err != nil {
		return err
	}
	for _, origin := range strings.Split(valueOrDefault(lookup, "TRUSTED_ORIGINS", ""), ",") {
		origin = strings.TrimSpace(origin)
		parsed, parseErr := url.Parse(origin)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("TRUSTED_ORIGINS must contain comma-separated HTTPS origins without paths")
		}
		config.TrustedOrigins = append(config.TrustedOrigins, origin)
	}
	for _, cidr := range strings.Split(valueOrDefault(lookup, "TRUSTED_PROXY_CIDRS", ""), ",") {
		cidr = strings.TrimSpace(cidr)
		if _, _, parseErr := net.ParseCIDR(cidr); parseErr != nil {
			return errors.New("TRUSTED_PROXY_CIDRS must contain comma-separated CIDR blocks")
		}
		config.TrustedProxyCIDRs = append(config.TrustedProxyCIDRs, cidr)
	}
	return nil
}

func valueOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok {
		return value
	}
	return fallback
}

func uint32Value(lookup func(string) (string, bool), key string, fallback uint32) (uint32, error) {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func requiredBase64(lookup func(string) (string, bool), key string, length int) ([]byte, error) {
	decoded, err := requiredBase64AtLeast(lookup, key, length)
	if err != nil {
		return nil, err
	}
	if len(decoded) != length {
		return nil, fmt.Errorf("%s must decode to exactly %d bytes", key, length)
	}
	return decoded, nil
}

func requiredBase64AtLeast(lookup func(string) (string, bool), key string, minimum int) ([]byte, error) {
	value := valueOrDefault(lookup, key, "")
	decoded, err := base64.RawStdEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) < minimum {
		return nil, fmt.Errorf("%s must be unpadded base64 containing at least %d bytes", key, minimum)
	}
	return decoded, nil
}
