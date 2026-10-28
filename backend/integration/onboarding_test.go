//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"barbershop-app/backend/internal/establishment"
)

func TestOwnerRegistrationAndEmailConfirmation(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	confirmationToken := registerOwner(t, application, registration)

	var persisted struct {
		Establishments int
		Users          int
		Credentials    int
		Confirmations  int
	}
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM establishments),
		(SELECT count(*) FROM users),
		(SELECT count(*) FROM user_credentials),
		(SELECT count(*) FROM email_confirmations)`).Scan(
		&persisted.Establishments, &persisted.Users, &persisted.Credentials, &persisted.Confirmations,
	); err != nil {
		t.Fatalf("query persisted registration: %v", err)
	}
	if persisted.Establishments != 1 || persisted.Users != 1 || persisted.Credentials != 1 || persisted.Confirmations != 1 {
		t.Fatalf("registration was not persisted completely: %+v", persisted)
	}

	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": registration.Email, "password": registration.Password,
	})
	assertStatus(t, response, http.StatusForbidden)
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": confirmationToken})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": confirmationToken})
	assertStatus(t, response, http.StatusBadRequest)

	var confirmedUsers, trials, sessions int
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM users WHERE email_confirmed_at IS NOT NULL),
		(SELECT count(*) FROM establishment_trials),
		(SELECT count(*) FROM auth_sessions)`).Scan(&confirmedUsers, &trials, &sessions); err != nil {
		t.Fatalf("query persisted confirmation: %v", err)
	}
	if confirmedUsers != 1 || trials != 1 || sessions != 1 {
		t.Fatalf("confirmation was not persisted completely: users=%d trials=%d sessions=%d",
			confirmedUsers, trials, sessions)
	}
}

func TestConfirmationReplacementAndRegistrationConflicts(t *testing.T) {
	application := newTestApplication(t)
	registration := ownerRegistration{
		EstablishmentName: "Primeira Barbearia",
		Slug:              "primeira-barbearia",
		TimeZone:          "America/Sao_Paulo",
		Email:             "first@example.test",
		Password:          "senha-segura",
	}
	oldToken := registerOwner(t, application, registration)

	response := application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation", map[string]string{"email": registration.Email})
	assertStatus(t, response, http.StatusAccepted)
	newToken := application.messages.latestToken(t)
	if newToken == oldToken {
		t.Fatal("confirmation request did not replace the active token")
	}
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": oldToken})
	assertStatus(t, response, http.StatusBadRequest)
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": newToken})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation", map[string]string{"email": "absent@example.test"})
	assertStatus(t, response, http.StatusAccepted)

	response = application.request(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"establishment_name": "Segunda Barbearia", "slug": "segunda-barbearia",
		"time_zone": "America/Sao_Paulo", "email": "FIRST@example.test", "password": "outra-senha",
	})
	assertStatus(t, response, http.StatusConflict)
	response = application.request(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"establishment_name": "Terceira Barbearia", "slug": registration.Slug,
		"time_zone": "America/Sao_Paulo", "email": "third@example.test", "password": "outra-senha",
	})
	assertStatus(t, response, http.StatusConflict)

	var establishments, users int
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM establishments), (SELECT count(*) FROM users)`).Scan(&establishments, &users); err != nil {
		t.Fatalf("count registrations after conflicts: %v", err)
	}
	if establishments != 1 || users != 1 {
		t.Fatalf("conflicting registrations left partial data: establishments=%d users=%d", establishments, users)
	}
}

func TestOwnerRegistrationRejectsReservedPlatformRouteSlug(t *testing.T) {
	application := newTestApplication(t)
	registration := ownerRegistration{
		EstablishmentName: "Barbearia em Rota Reservada",
		Slug:              "cadastro",
		TimeZone:          "America/Sao_Paulo",
		Email:             "reserved-route@example.test",
		Password:          "senha-segura",
	}
	response := application.request(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"establishment_name": registration.EstablishmentName,
		"slug":               registration.Slug,
		"time_zone":          registration.TimeZone,
		"email":              registration.Email,
		"password":           registration.Password,
	})
	assertStatus(t, response, http.StatusBadRequest)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read reserved slug response: %v", err)
	}
	if !strings.Contains(string(body), `"field":"slug"`) || !strings.Contains(string(body), `"code":"reserved_slug"`) {
		t.Fatalf("reserved slug error did not identify the slug field: %s", body)
	}
	var establishments, users int
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM establishments), (SELECT count(*) FROM users)`).Scan(&establishments, &users); err != nil {
		t.Fatalf("count registrations after reserved slug: %v", err)
	}
	if establishments != 0 || users != 0 {
		t.Fatalf("reserved slug registration left partial data: establishments=%d users=%d", establishments, users)
	}
}

func TestLegacyReservedSlugRemainsUnchangedAndIsFlaggedAsPublicConflict(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	const legacySlug = "painel"
	var establishmentID string
	if err := integrationPool.QueryRow(ctx, `
		INSERT INTO establishments (name, slug, time_zone)
		VALUES ('Barbearia Legada', $1, 'America/Sao_Paulo')
		RETURNING id::text
	`, legacySlug).Scan(&establishmentID); err != nil {
		t.Fatalf("insert legacy establishment with route-conflicting slug: %v", err)
	}
	var storedSlug string
	if err := integrationPool.QueryRow(ctx, "SELECT slug FROM establishments WHERE id = $1::uuid", establishmentID).Scan(&storedSlug); err != nil {
		t.Fatalf("read legacy establishment slug: %v", err)
	}
	if storedSlug != legacySlug {
		t.Fatalf("legacy slug changed from %q to %q", legacySlug, storedSlug)
	}
	if !establishment.IsPublicSlugReserved(storedSlug) {
		t.Fatalf("legacy slug %q was not flagged as a public route conflict", storedSlug)
	}
}
