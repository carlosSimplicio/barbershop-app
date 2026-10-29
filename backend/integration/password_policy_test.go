//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"barbershop-app/backend/internal/auth/security"
)

func TestPasswordPolicyRejectsInvisibleCharactersAcrossCredentialFlows(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	response := application.request(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"establishment_name": registration.EstablishmentName, "slug": registration.Slug,
		"time_zone": registration.TimeZone, "email": registration.Email, "password": "abcd\uFEFFefgh",
	})
	assertPasswordPolicyError(t, response, "password")
	var users int
	if err := integrationPool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatal("invalid registration persisted a user")
	}
	registerAndConfirmOwner(t, application, registration)
	response = application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Profissional de teste", "email": "professional@example.test", "temporary_password": "abcd\u200Befgh",
	})
	assertPasswordPolicyError(t, response, "temporary_password")
	if err := integrationPool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatal("invalid temporary password persisted a professional")
	}
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": registration.Email, "password": "abcd\u0000efgh",
	})
	assertPasswordPolicyError(t, response, "password")
	token := requestRecoveryToken(t, application, registration.Email)
	response = application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{
		"token": token, "new_password": "abcd\u034Fefgh",
	})
	assertPasswordPolicyError(t, response, "new_password")
	var consumed bool
	if err := integrationPool.QueryRow(context.Background(), "SELECT consumed_at IS NOT NULL FROM password_recovery_tokens WHERE token_hash=$1", security.HashOpaqueToken(token)).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed {
		t.Fatal("invalid new password consumed the recovery token")
	}
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	createProfessionalForList(t, application, "Profissional de teste", "professional@example.test")
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "professional@example.test", "password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "abcd\uFE0Fefgh"})
	assertPasswordPolicyError(t, response, "new_password")
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var principal struct {
		PasswordChangeRequired bool `json:"password_change_required"`
	}
	decodeJSON(t, response, &principal)
	if !principal.PasswordChangeRequired {
		t.Fatal("invalid password completed the mandatory change")
	}
}

func assertPasswordPolicyError(t *testing.T, response *http.Response, field string) {
	t.Helper()
	assertStatus(t, response, http.StatusBadRequest)
	var failure struct {
		Code   string `json:"code"`
		Fields []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"fields"`
	}
	decodeJSON(t, response, &failure)
	if failure.Code != "validation_failed" || len(failure.Fields) != 1 || failure.Fields[0].Field != field || failure.Fields[0].Code != "invalid_password" {
		t.Fatal("invalid password did not produce the documented field error")
	}
}
