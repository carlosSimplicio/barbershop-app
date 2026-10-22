package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type Principal struct {
	UserID            string
	SessionID         string
	EstablishmentID   string
	Role              string
	CredentialVersion int64
}

type AccessClaims struct {
	Subject           string `json:"sub"`
	SessionID         string `json:"session_id"`
	EstablishmentID   string `json:"establishment_id"`
	Role              string `json:"role"`
	ExpiresAt         int64  `json:"exp"`
	CredentialVersion int64  `json:"credential_version"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

type JWTManager struct {
	activeKeyID string
	privateKey  ed25519.PrivateKey
	publicKeys  map[string]ed25519.PublicKey
	lifetime    time.Duration
}

func NewJWTManager(activeKeyID string, privateKey ed25519.PrivateKey, publicKeys map[string]ed25519.PublicKey, lifetime time.Duration) (*JWTManager, error) {
	if activeKeyID == "" || len(privateKey) != ed25519.PrivateKeySize || lifetime <= 0 {
		return nil, errors.New("invalid JWT signing configuration")
	}
	activePublicKey, ok := publicKeys[activeKeyID]
	if !ok || len(activePublicKey) != ed25519.PublicKeySize || !privateKey.Public().(ed25519.PublicKey).Equal(activePublicKey) {
		return nil, errors.New("active JWT public key does not match private key")
	}
	keys := make(map[string]ed25519.PublicKey, len(publicKeys))
	for keyID, key := range publicKeys {
		if keyID == "" || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("invalid JWT verification key")
		}
		keys[keyID] = append(ed25519.PublicKey(nil), key...)
	}
	return &JWTManager{activeKeyID: activeKeyID, privateKey: append(ed25519.PrivateKey(nil), privateKey...), publicKeys: keys, lifetime: lifetime}, nil
}

func (manager *JWTManager) Issue(principal Principal, now time.Time) (string, error) {
	if principal.UserID == "" || principal.SessionID == "" || principal.EstablishmentID == "" ||
		(principal.Role != "owner" && principal.Role != "professional") || principal.CredentialVersion <= 0 {
		return "", errors.New("invalid access token principal")
	}
	header, err := json.Marshal(jwtHeader{Algorithm: "EdDSA", Type: "JWT", KeyID: manager.activeKeyID})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(AccessClaims{
		Subject: principal.UserID, SessionID: principal.SessionID, EstablishmentID: principal.EstablishmentID, Role: principal.Role,
		ExpiresAt: now.Add(manager.lifetime).Unix(), CredentialVersion: principal.CredentialVersion,
	})
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claims)
	unsigned := encodedHeader + "." + encodedClaims
	signature := ed25519.Sign(manager.privateKey, []byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (manager *JWTManager) Verify(token string, now time.Time) (AccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	headerBytes, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "EdDSA" || header.Type != "JWT" || header.KeyID == "" {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	publicKey, ok := manager.publicKeys[header.KeyID]
	if !ok {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	signature, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	claimsBytes, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	var claims AccessClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil || claims.Subject == "" || claims.SessionID == "" || claims.EstablishmentID == "" ||
		(claims.Role != "owner" && claims.Role != "professional") || claims.CredentialVersion <= 0 || claims.ExpiresAt <= now.Unix() {
		return AccessClaims{}, ErrInvalidAccessToken
	}
	return claims, nil
}
