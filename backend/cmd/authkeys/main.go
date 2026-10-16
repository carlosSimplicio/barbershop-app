package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "generate authentication keys:", err)
		os.Exit(1)
	}
}

func run() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	csrfKey := make([]byte, 32)
	if _, err := rand.Read(csrfKey); err != nil {
		return err
	}
	rateLimitKey := make([]byte, 32)
	if _, err := rand.Read(rateLimitKey); err != nil {
		return err
	}
	publicKeys, err := json.Marshal(map[string]string{
		"local-1": base64.RawStdEncoding.EncodeToString(publicKey),
	})
	if err != nil {
		return err
	}

	fmt.Println("JWT_ACTIVE_KEY_ID=local-1")
	fmt.Println("JWT_PRIVATE_KEY_BASE64=" + base64.RawStdEncoding.EncodeToString(privateKey.Seed()))
	fmt.Println("JWT_PUBLIC_KEYS_JSON=" + string(publicKeys))
	fmt.Println("CSRF_SIGNING_KEY_BASE64=" + base64.RawStdEncoding.EncodeToString(csrfKey))
	fmt.Println("RATE_LIMIT_HMAC_KEY_BASE64=" + base64.RawStdEncoding.EncodeToString(rateLimitKey))
	return nil
}
