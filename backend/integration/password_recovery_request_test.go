//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"
)

func TestPasswordRecoveryRequestReplacesTokenForConfirmedOwner(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	confirmationToken := registerOwner(t, application, registration)

	response := application.request(t, http.MethodPost, "/api/v1/auth/password/recovery", map[string]string{"email": registration.Email})
	assertStatus(t, response, http.StatusAccepted)
	if len(application.messages.messages) != 1 {
		t.Fatalf("unconfirmed owner received recovery mail: %d messages", len(application.messages.messages))
	}
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": confirmationToken})
	assertStatus(t, response, http.StatusOK)

	response = application.request(t, http.MethodPost, "/api/v1/auth/password/recovery", map[string]string{"email": strings.ToUpper(registration.Email)})
	assertStatus(t, response, http.StatusAccepted)
	firstToken := application.messages.latestToken(t)
	firstMessage := application.messages.messages[len(application.messages.messages)-1]
	if !strings.Contains(firstMessage.Text, trustedOrigin+"/redefinir-senha#token="+firstToken) || strings.Contains(firstMessage.Text, "?token=") {
		t.Fatal("recovery URL did not keep the token in its fragment")
	}
	var firstHash []byte
	var firstExpiry time.Time
	if err := integrationPool.QueryRow(context.Background(), `SELECT token_hash, expires_at FROM password_recovery_tokens WHERE invalidated_at IS NULL`).Scan(&firstHash, &firstExpiry); err != nil {
		t.Fatalf("load first recovery token: %v", err)
	}
	if !bytes.Equal(firstHash, security.HashOpaqueToken(firstToken)) || bytes.Equal(firstHash, []byte(firstToken)) || time.Until(firstExpiry) < 59*time.Minute || time.Until(firstExpiry) > time.Hour {
		t.Fatal("recovery token was not stored as a one-hour hash")
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/password/recovery", map[string]string{"email": registration.Email})
	assertStatus(t, response, http.StatusAccepted)
	secondToken := application.messages.latestToken(t)
	if secondToken == firstToken {
		t.Fatal("recovery token was reused")
	}
	var previousInvalidated bool
	var activeCount int
	if err := integrationPool.QueryRow(context.Background(), `SELECT invalidated_at IS NOT NULL FROM password_recovery_tokens WHERE token_hash = $1`, firstHash).Scan(&previousInvalidated); err != nil {
		t.Fatalf("load previous recovery state: %v", err)
	}
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM password_recovery_tokens WHERE invalidated_at IS NULL AND consumed_at IS NULL`).Scan(&activeCount); err != nil {
		t.Fatalf("count active recovery tokens: %v", err)
	}
	if !previousInvalidated || activeCount != 1 {
		t.Fatalf("previous invalidated = %t, active count = %d", previousInvalidated, activeCount)
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET role = 'professional' WHERE normalized_email = $1`, registration.Email); err != nil {
		t.Fatalf("change owner role: %v", err)
	}
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/recovery", map[string]string{"email": registration.Email})
	assertStatus(t, response, http.StatusAccepted)
	if len(application.messages.messages) != 3 {
		t.Fatal("professional received a recovery email")
	}
}

func TestPasswordRecoveryRequestDoesNotRevealMissingAccountAndLimitsAttempts(t *testing.T) {
	application := newTestApplication(t)
	path := "/api/v1/auth/password/recovery"
	response := application.request(t, http.MethodPost, path, map[string]string{"email": "missing@example.test"})
	assertStatus(t, response, http.StatusAccepted)
	if len(application.messages.messages) != 0 {
		t.Fatal("missing account received recovery mail")
	}
	for attempt := 0; attempt < 4; attempt++ {
		response = application.request(t, http.MethodPost, path, map[string]string{"email": "missing@example.test"})
		assertStatus(t, response, http.StatusAccepted)
	}
	response = application.request(t, http.MethodPost, path, map[string]string{"email": "missing@example.test"})
	assertStatus(t, response, http.StatusTooManyRequests)
	if response.Header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	response = application.request(t, http.MethodPost, path, map[string]string{"email": "invalid"})
	assertStatus(t, response, http.StatusBadRequest)
}
