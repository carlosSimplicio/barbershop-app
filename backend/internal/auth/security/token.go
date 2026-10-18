package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

func NewOpaqueToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func HashOpaqueToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

type HMACSigner struct {
	key []byte
}

func NewHMACSigner(key []byte) (*HMACSigner, error) {
	if len(key) < 32 {
		return nil, errors.New("HMAC key must contain at least 32 bytes")
	}
	return &HMACSigner{key: append([]byte(nil), key...)}, nil
}

func (signer *HMACSigner) Sign(value string) string {
	mac := hmac.New(sha256.New, signer.key)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (signer *HMACSigner) Key(dimension, value string) []byte {
	mac := hmac.New(sha256.New, signer.key)
	_, _ = mac.Write([]byte(dimension))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (signer *HMACSigner) NewSignedToken() (string, error) {
	nonce, err := NewOpaqueToken()
	if err != nil {
		return "", err
	}
	return nonce + "." + signer.Sign(nonce), nil
}

func (signer *HMACSigner) VerifySignedToken(token string) bool {
	nonce, signature, found := strings.Cut(token, ".")
	if !found || nonce == "" || signature == "" {
		return false
	}
	expected := signer.Sign(nonce)
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) == 1
}
