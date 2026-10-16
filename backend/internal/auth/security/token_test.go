package security

import (
	"bytes"
	"testing"
)

func TestOpaqueTokensAreRandomAndHashable(t *testing.T) {
	first, err := NewOpaqueToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	second, err := NewOpaqueToken()
	if err != nil {
		t.Fatalf("new second token: %v", err)
	}
	if first == second || bytes.Equal(HashOpaqueToken(first), HashOpaqueToken(second)) {
		t.Fatal("expected independent opaque tokens")
	}
}

func TestHMACSignerSignsTokensAndKeysWithDomainSeparation(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	signer, err := NewHMACSigner(key)
	if err != nil {
		t.Fatalf("new HMAC signer: %v", err)
	}
	token, err := signer.NewSignedToken()
	if err != nil {
		t.Fatalf("new signed token: %v", err)
	}
	if !signer.VerifySignedToken(token) || signer.VerifySignedToken(token+"x") {
		t.Fatal("signed token verification did not distinguish tampering")
	}
	if bytes.Equal(signer.Key("ip", "value"), signer.Key("email", "value")) {
		t.Fatal("expected dimension-separated rate limit keys")
	}
}
