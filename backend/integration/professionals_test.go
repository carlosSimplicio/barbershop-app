//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOwnerCreatesProfessionalAndProfessionalMustChangePassword(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())

	response := application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Ana Silva", "email": "ana@example.test", "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusCreated)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read created professional: %v", err)
	}
	if bytes.Contains(body, []byte("senha-temporaria")) || bytes.Contains(body, []byte("temporary_password")) || bytes.Contains(body, []byte("password_hash")) {
		t.Fatal("professional response exposed credential material")
	}
	var created struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Active bool   `json:"active"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created professional: %v", err)
	}
	if created.ID == "" || created.Name != "Ana Silva" || created.Email != "ana@example.test" || !created.Active {
		t.Fatalf("created professional = %#v", created)
	}
	var users, credentials, profiles int
	var sameEstablishment bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM users WHERE role = 'professional'),
		(SELECT count(*) FROM user_credentials c JOIN users u ON u.id = c.user_id WHERE u.role = 'professional'),
		(SELECT count(*) FROM professionals),
		(SELECT p.establishment_id = owner.establishment_id FROM professionals p
		 JOIN users u ON u.id = p.user_id CROSS JOIN users owner
		 WHERE u.normalized_email = 'ana@example.test' AND owner.normalized_email = 'owner@example.test')`).Scan(
		&users, &credentials, &profiles, &sameEstablishment,
	); err != nil {
		t.Fatalf("count created professional: %v", err)
	}
	if users != 1 || credentials != 1 || profiles != 1 || !sameEstablishment {
		t.Fatalf("incomplete or cross-tenant professional: users=%d credentials=%d profiles=%d same_establishment=%v", users, credentials, profiles, sameEstablishment)
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "ana@example.test", "password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusOK)
	var principal struct {
		Role                   string `json:"role"`
		PasswordChangeRequired bool   `json:"password_change_required"`
	}
	if err := json.NewDecoder(response.Body).Decode(&principal); err != nil {
		t.Fatalf("decode professional login: %v", err)
	}
	if principal.Role != "professional" || !principal.PasswordChangeRequired {
		t.Fatalf("professional login = %#v", principal)
	}
	response = application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Bia", "email": "bia@example.test", "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusForbidden)
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Bia", "email": "bia@example.test", "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusForbidden)
}

func TestProfessionalRegistrationRejectsInvalidFields(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	tests := []struct {
		name  string
		body  map[string]string
		field string
	}{
		{name: "blank name", body: map[string]string{"name": "  ", "email": "ana@example.test", "temporary_password": "senha-temporaria"}, field: "name"},
		{name: "long name", body: map[string]string{"name": strings.Repeat("a", 121), "email": "ana@example.test", "temporary_password": "senha-temporaria"}, field: "name"},
		{name: "invalid email", body: map[string]string{"name": "Ana", "email": "invalid", "temporary_password": "senha-temporaria"}, field: "email"},
		{name: "invalid password", body: map[string]string{"name": "Ana", "email": "ana@example.test", "temporary_password": "short"}, field: "temporary_password"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := application.request(t, http.MethodPost, "/api/v1/professionals", test.body)
			assertStatus(t, response, http.StatusBadRequest)
			assertProfessionalFieldError(t, response, test.field)
		})
	}
	var professionals int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM professionals`).Scan(&professionals); err != nil {
		t.Fatal(err)
	}
	if professionals != 0 {
		t.Fatalf("invalid requests created %d professionals", professionals)
	}
}

func TestProfessionalRegistrationRequiresSessionOriginAndCSRF(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	body := map[string]string{"name": "Ana", "email": "ana@example.test", "temporary_password": "senha-temporaria"}

	for _, test := range []struct {
		name   string
		origin string
		csrf   string
	}{
		{name: "missing origin", csrf: application.cookie(t, csrfCookieName).Value},
		{name: "missing csrf", origin: trustedOrigin},
		{name: "invalid csrf", origin: trustedOrigin, csrf: "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := requestProfessionalWithHeaders(t, application, body, test.origin, test.csrf)
			assertStatus(t, response, http.StatusForbidden)
		})
	}

	cookieURL, err := url.Parse(application.server.URL + "/api/v1/professionals")
	if err != nil {
		t.Fatal(err)
	}
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Path: "/api/v1", MaxAge: -1, Secure: true}})
	response := application.request(t, http.MethodPost, "/api/v1/professionals", body)
	assertStatus(t, response, http.StatusUnauthorized)
	var professionals int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM professionals`).Scan(&professionals); err != nil {
		t.Fatal(err)
	}
	if professionals != 0 {
		t.Fatalf("unauthorized requests created %d professionals", professionals)
	}
}

func TestProfessionalRegistrationRejectsExpiredTrialAndGloballyDuplicateEmail(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	body := map[string]string{"name": "Ana", "email": "ana@example.test", "temporary_password": "senha-temporaria"}
	response := application.request(t, http.MethodPost, "/api/v1/professionals", body)
	assertStatus(t, response, http.StatusCreated)

	second := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, second)
	response = application.request(t, http.MethodPost, "/api/v1/professionals", body)
	assertStatus(t, response, http.StatusConflict)
	assertProfessionalFieldError(t, response, "email")

	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2 WHERE establishment_id = (
		SELECT establishment_id FROM users WHERE normalized_email = 'other@example.test')`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("expire second trial: %v", err)
	}
	response = application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Bia", "email": "bia@example.test", "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusForbidden)
	var users, profiles int
	if err := integrationPool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM users WHERE normalized_email = 'bia@example.test'),
		(SELECT count(*) FROM professionals)`).Scan(&users, &profiles); err != nil {
		t.Fatal(err)
	}
	if users != 0 || profiles != 1 {
		t.Fatalf("rejected registrations persisted users=%d profiles=%d", users, profiles)
	}
}

func TestProfessionalRegistrationRollsBackWhenProfileWriteFails(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	ctx := context.Background()
	if _, err := integrationPool.Exec(ctx, `CREATE FUNCTION integration_reject_professional_profile() RETURNS trigger
		LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected profile failure'; END; $$`); err != nil {
		t.Fatalf("create failure injection: %v", err)
	}
	if _, err := integrationPool.Exec(ctx, `CREATE TRIGGER integration_reject_professional_profile
		BEFORE INSERT ON professionals FOR EACH ROW EXECUTE FUNCTION integration_reject_professional_profile()`); err != nil {
		t.Fatalf("install failure injection: %v", err)
	}
	t.Cleanup(func() {
		_, _ = integrationPool.Exec(ctx, `DROP TRIGGER IF EXISTS integration_reject_professional_profile ON professionals`)
		_, _ = integrationPool.Exec(ctx, `DROP FUNCTION IF EXISTS integration_reject_professional_profile()`)
	})

	response := application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": "Ana", "email": "ana@example.test", "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusInternalServerError)
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(responseBody, []byte("senha-temporaria")) || bytes.Contains(responseBody, []byte("injected profile failure")) {
		t.Fatal("error response exposed credential or database details")
	}
	var users, credentials, profiles int
	if err := integrationPool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM users WHERE role = 'professional'),
		(SELECT count(*) FROM user_credentials c JOIN users u ON u.id = c.user_id WHERE u.role = 'professional'),
		(SELECT count(*) FROM professionals)`).Scan(&users, &credentials, &profiles); err != nil {
		t.Fatal(err)
	}
	if users != 0 || credentials != 0 || profiles != 0 {
		t.Fatalf("failed transaction persisted users=%d credentials=%d profiles=%d", users, credentials, profiles)
	}
}

func assertProfessionalFieldError(t *testing.T, response *http.Response, field string) {
	t.Helper()
	var body struct {
		Fields []struct {
			Field string `json:"field"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode field error: %v", err)
	}
	if len(body.Fields) != 1 || body.Fields[0].Field != field {
		t.Fatalf("field errors = %#v, want %s", body.Fields, field)
	}
}

func requestProfessionalWithHeaders(t *testing.T, application *testApplication, body any, origin, csrf string) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, application.server.URL+"/api/v1/professionals", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	response, err := application.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}
