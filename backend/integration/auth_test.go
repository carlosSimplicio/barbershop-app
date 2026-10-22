//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"

	"github.com/google/uuid"
)

func TestOwnerSessionLifecycle(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	firstAccess := application.cookie(t, accessCookieName).Value
	firstCSRF := application.cookie(t, csrfCookieName).Value

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

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": registration.Email, "password": registration.Password,
	})
	assertStatus(t, response, http.StatusOK)
	if len(response.Cookies()) != 2 {
		t.Fatalf("login cookies = %d, want access and CSRF", len(response.Cookies()))
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == accessCookieName && cookie.MaxAge != 15*24*60*60 {
			t.Fatalf("access cookie Max-Age = %d, want 15 days", cookie.MaxAge)
		}
	}
	if application.cookie(t, accessCookieName).Value == firstAccess {
		t.Fatal("new login reused the prior session access token")
	}
	secondAccess := application.cookie(t, accessCookieName).Value
	response = application.request(t, http.MethodPost, "/api/v1/auth/refresh", nil)
	assertStatus(t, response, http.StatusNotFound)

	response = application.request(t, http.MethodPost, "/api/v1/auth/logout", nil)
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)
	cookieURL, err := url.Parse(application.server.URL + "/api/v1/auth/logout")
	if err != nil {
		t.Fatal(err)
	}
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: secondAccess, Path: "/api/v1", Secure: true}})
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{
		{Name: accessCookieName, Value: firstAccess, Path: "/api/v1", Secure: true},
		{Name: csrfCookieName, Value: firstCSRF, Path: "/", Secure: true},
	})
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/logout", nil)
	assertStatus(t, response, http.StatusNoContent)
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: firstAccess, Path: "/api/v1", Secure: true}})
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "OWNER@example.test", "password": registration.Password,
	})
	assertStatus(t, response, http.StatusOK)
}

func TestAccessClaimsMustMatchActiveSession(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	access := application.cookie(t, accessCookieName).Value
	claims, err := application.accessTokens.Verify(access, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var initialExpiry time.Time
	if err := integrationPool.QueryRow(context.Background(), `SELECT expires_at FROM auth_sessions WHERE id = $1`, claims.SessionID).Scan(&initialExpiry); err != nil {
		t.Fatal(err)
	}
	cookieURL, err := url.Parse(application.server.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	base := security.Principal{
		UserID: claims.Subject, SessionID: claims.SessionID, EstablishmentID: claims.EstablishmentID,
		Role: claims.Role, CredentialVersion: claims.CredentialVersion,
	}
	for _, tc := range []struct {
		name      string
		principal security.Principal
		issuedAt  time.Time
	}{
		{name: "other tenant", principal: func() security.Principal { p := base; p.EstablishmentID = uuid.NewString(); return p }(), issuedAt: time.Now()},
		{name: "other role", principal: func() security.Principal { p := base; p.Role = "professional"; return p }(), issuedAt: time.Now()},
		{name: "other user", principal: func() security.Principal { p := base; p.UserID = uuid.NewString(); return p }(), issuedAt: time.Now()},
		{name: "unknown session", principal: func() security.Principal { p := base; p.SessionID = uuid.NewString(); return p }(), issuedAt: time.Now()},
		{name: "expired access", principal: base, issuedAt: time.Now().Add(-15*24*time.Hour - time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forged, err := application.accessTokens.Issue(tc.principal, tc.issuedAt)
			if err != nil {
				t.Fatal(err)
			}
			application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: forged, Path: "/api/v1", Secure: true}})
			response := application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
			assertStatus(t, response, http.StatusUnauthorized)
		})
	}
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: access, Path: "/api/v1", Secure: true}})
	response := application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var expiryAfterUse time.Time
	if err := integrationPool.QueryRow(context.Background(), `SELECT expires_at FROM auth_sessions WHERE id = $1`, claims.SessionID).Scan(&expiryAfterUse); err != nil {
		t.Fatal(err)
	}
	if !expiryAfterUse.Equal(initialExpiry) {
		t.Fatalf("session expiry changed after use: before=%s after=%s", initialExpiry, expiryAfterUse)
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE auth_sessions SET created_at = now() - interval '2 days', expires_at = now() - interval '1 day' WHERE id = $1`, claims.SessionID); err != nil {
		t.Fatal(err)
	}
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)
}

func TestLogoutWithoutValidAccessClearsCookies(t *testing.T) {
	for _, state := range []string{"missing", "invalid", "expired"} {
		t.Run(state, func(t *testing.T) {
			application := newTestApplication(t)
			registerAndConfirmOwner(t, application, defaultOwnerRegistration())
			originalAccess := application.cookie(t, accessCookieName).Value
			cookieURL, err := url.Parse(application.server.URL + "/api/v1/auth/logout")
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "missing":
				application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Path: "/api/v1", MaxAge: -1}})
			case "invalid":
				application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: "invalid", Path: "/api/v1", Secure: true}})
			case "expired":
				claims, err := application.accessTokens.Verify(originalAccess, time.Now())
				if err != nil {
					t.Fatal(err)
				}
				expired, err := application.accessTokens.Issue(security.Principal{
					UserID: claims.Subject, SessionID: claims.SessionID, EstablishmentID: claims.EstablishmentID,
					Role: claims.Role, CredentialVersion: claims.CredentialVersion,
				}, time.Now().Add(-15*24*time.Hour-time.Minute))
				if err != nil {
					t.Fatal(err)
				}
				application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: expired, Path: "/api/v1", Secure: true}})
			}
			response := application.request(t, http.MethodPost, "/api/v1/auth/logout", nil)
			assertStatus(t, response, http.StatusNoContent)
			if len(application.client.Jar.Cookies(cookieURL)) != 0 {
				t.Fatalf("logout left cookies in browser jar: %#v", application.client.Jar.Cookies(cookieURL))
			}
			application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: originalAccess, Path: "/api/v1", Secure: true}})
			response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
			assertStatus(t, response, http.StatusOK)
		})
	}
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
	oldAccess := application.cookie(t, accessCookieName).Value
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodGet, "/api/v1/establishment", nil)
	assertStatus(t, response, http.StatusForbidden)

	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": temporaryPassword})
	assertStatus(t, response, http.StatusBadRequest)
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"})
	assertStatus(t, response, http.StatusNoContent)
	newAccess := application.cookie(t, accessCookieName).Value
	if newAccess == oldAccess {
		t.Fatal("password change reused the prior access token")
	}
	cookieURL, err := url.Parse(application.server.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: oldAccess, Path: "/api/v1", Secure: true}})
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)
	application.client.Jar.SetCookies(cookieURL, []*http.Cookie{{Name: accessCookieName, Value: newAccess, Path: "/api/v1", Secure: true}})

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
