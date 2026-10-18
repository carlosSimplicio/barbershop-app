package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestJWTManagerIssuesAndVerifiesAccessToken(t *testing.T) {
	manager := testJWTManager(t)
	now := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)
	token, err := manager.Issue(Principal{UserID: "user-1", EstablishmentID: "shop-1", Role: "owner", CredentialVersion: 3}, now)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	claims, err := manager.Verify(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Subject != "user-1" || claims.EstablishmentID != "shop-1" || claims.Role != "owner" || claims.CredentialVersion != 3 {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestJWTManagerRejectsTamperingUnknownKeyAndExpiry(t *testing.T) {
	manager := testJWTManager(t)
	now := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)
	token, err := manager.Issue(Principal{UserID: "user-1", EstablishmentID: "shop-1", Role: "professional", CredentialVersion: 1}, now)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	parts := strings.Split(token, ".")
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"attacker"}`))
	if _, err := manager.Verify(parts[0]+"."+tamperedPayload+"."+parts[2], now); err == nil {
		t.Fatal("expected tampered token to fail")
	}
	if _, err := manager.Verify(token, now.Add(16*time.Minute)); err == nil {
		t.Fatal("expected expired token to fail")
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT","kid":"unknown"}`))
	if _, err := manager.Verify(header+"."+parts[1]+"."+parts[2], now); err == nil {
		t.Fatal("expected unknown key to fail")
	}
}

func testJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	manager, err := NewJWTManager("key-1", privateKey, map[string]ed25519.PublicKey{
		"key-1": privateKey.Public().(ed25519.PublicKey),
	}, 15*time.Minute)
	if err != nil {
		t.Fatalf("new JWT manager: %v", err)
	}
	return manager
}
