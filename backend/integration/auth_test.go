//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestOwnerSessionLifecycle(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	oldRefreshToken := application.cookie(t, refreshCookieName).Value

	response := application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var principal struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	decodeJSON(t, response, &principal)
	if principal.Email != registration.Email || principal.Role != "owner" {
		t.Fatalf("unexpected authenticated principal: %+v", principal)
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/refresh", nil)
	assertStatus(t, response, http.StatusNoContent)
	if rotated := application.cookie(t, refreshCookieName).Value; rotated == oldRefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/logout", nil)
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "OWNER@example.test", "password": registration.Password,
	})
	assertStatus(t, response, http.StatusOK)
}

func TestProfessionalMustChangeTemporaryPassword(t *testing.T) {
	application := newTestApplication(t)
	establishmentID := uuid.New()
	professionalID := uuid.New()
	temporaryPassword := "senha-temporaria"
	hash, err := application.passwords.Hash(temporaryPassword)
	if err != nil {
		t.Fatalf("hash temporary password: %v", err)
	}
	ctx := context.Background()
	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin professional seed: %v", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `INSERT INTO establishments (id, name, slug, time_zone)
		VALUES ($1, 'Estúdio Teste', 'estudio-teste', 'America/Sao_Paulo')`, establishmentID); err != nil {
		t.Fatalf("seed professional establishment: %v", err)
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO users
		(id, establishment_id, email, normalized_email, role, active, password_change_required)
		VALUES ($1, $2, 'professional@example.test', 'professional@example.test', 'professional', true, true)`,
		professionalID, establishmentID); err != nil {
		t.Fatalf("seed professional user: %v", err)
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO user_credentials (user_id, password_hash) VALUES ($1, $2)`, professionalID, hash); err != nil {
		t.Fatalf("seed professional: %v", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("commit professional seed: %v", err)
	}

	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "professional@example.test", "password": temporaryPassword,
	})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodGet, "/api/v1/establishment", nil)
	assertStatus(t, response, http.StatusForbidden)

	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": temporaryPassword})
	assertStatus(t, response, http.StatusBadRequest)
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"})
	assertStatus(t, response, http.StatusNoContent)

	response = application.request(t, http.MethodGet, "/api/v1/establishment", nil)
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/logout", nil)
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "professional@example.test", "password": temporaryPassword,
	})
	assertStatus(t, response, http.StatusUnauthorized)
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "professional@example.test", "password": "senha-definitiva",
	})
	assertStatus(t, response, http.StatusOK)
}
