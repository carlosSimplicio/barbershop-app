package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"

	"github.com/gin-gonic/gin"
)

type accessVerifierStub struct {
	claims security.AccessClaims
	err    error
}

func (stub accessVerifierStub) Verify(string, time.Time) (security.AccessClaims, error) {
	return stub.claims, stub.err
}

type userStoreStub struct {
	user User
	err  error
}

func (stub userStoreStub) FindActiveSessionUser(context.Context, string, time.Time) (User, error) {
	return stub.user, stub.err
}

func TestMiddlewareRejectsUnsafeRequestFromUnknownOrigin(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/login", nil, map[string]string{"Origin": "https://evil.example"})
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestMiddlewareAllowsPublicRegistrationFromTrustedOriginWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/register", nil, map[string]string{"Origin": "https://localhost:8443"})
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
}

func TestMiddlewareAllowsPublicEmailConfirmationRequestFromTrustedOriginWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/email-confirmation", nil, map[string]string{"Origin": "https://localhost:8443"})
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
}

func TestMiddlewareAllowsPublicPasswordRecoveryRequestFromTrustedOriginWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/password/recovery", nil, map[string]string{"Origin": "https://localhost:8443"})
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
	status = runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/password/recovery", nil, map[string]string{"Origin": "https://evil.example"})
	if status != http.StatusForbidden {
		t.Fatalf("untrusted origin status = %d, want 403", status)
	}
}

func TestMiddlewareAllowsPublicPasswordRecoveryVerificationFromTrustedOriginWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/password/recovery/verify", nil, map[string]string{"Origin": "https://localhost:8443"})
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
	status = runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/password/recovery/verify", nil, map[string]string{"Origin": "https://evil.example"})
	if status != http.StatusForbidden {
		t.Fatalf("untrusted origin status = %d, want 403", status)
	}
}

func TestMiddlewareAllowsPublicEmailConfirmationVerificationFromTrustedOriginWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/email-confirmation/verify", nil, map[string]string{"Origin": "https://localhost:8443"})
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
}

func TestMiddlewareRequiresMatchingSignedCSRFToken(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	headers := map[string]string{"Origin": "https://localhost:8443", "X-CSRF-Token": "different"}
	status := runMiddlewareRequest(t, middleware, http.MethodPost, "/api/v1/auth/logout", []*http.Cookie{{Name: csrfCookieName, Value: "cookie"}}, headers)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestMiddlewareRequiresMatchingSignedCSRFTokenForEstablishmentSettingsPatch(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	headers := map[string]string{"Origin": "https://localhost:8443", "X-CSRF-Token": "different"}
	status := runMiddlewareRequest(t, middleware, http.MethodPatch, "/api/v1/establishment/settings", []*http.Cookie{{Name: csrfCookieName, Value: "cookie"}}, headers)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestMiddlewareDerivesTenantAndRoleFromCurrentUser(t *testing.T) {
	user := validSessionUser()
	claims := security.AccessClaims{
		Subject: user.ID, SessionID: "session-1", EstablishmentID: user.EstablishmentID, Role: user.Role,
		CredentialVersion: user.CredentialVersion, ExpiresAt: testNow.Add(time.Minute).Unix(),
	}
	middleware := testMiddleware(t, user, claims)
	status := runMiddlewareRequest(t, middleware, http.MethodGet, "/api/v1/auth/me", []*http.Cookie{{Name: accessCookieName, Value: "access"}}, nil)
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", status)
	}
}

func TestMiddlewareRejectsEstablishmentRequestWithoutSession(t *testing.T) {
	middleware := testMiddleware(t, validSessionUser(), security.AccessClaims{})
	status := runMiddlewareRequest(t, middleware, http.MethodGet, "/api/v1/establishment", nil, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestMiddlewareDistinguishesUnavailableSessionStoreFromInvalidSession(t *testing.T) {
	user := validSessionUser()
	claims := security.AccessClaims{
		Subject: user.ID, SessionID: "session-1", EstablishmentID: user.EstablishmentID, Role: user.Role,
		CredentialVersion: user.CredentialVersion, ExpiresAt: testNow.Add(time.Minute).Unix(),
	}
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "expired or revoked session", err: ErrInvalidSession, status: http.StatusUnauthorized},
		{name: "database unavailable", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			middleware := testMiddleware(t, user, claims)
			middleware.users = userStoreStub{err: tc.err}
			status := runMiddlewareRequest(t, middleware, http.MethodGet, "/api/v1/auth/me", []*http.Cookie{{Name: accessCookieName, Value: "access"}}, nil)
			if status != tc.status {
				t.Fatalf("status = %d, want %d", status, tc.status)
			}
		})
	}
}

func TestMiddlewareRejectsTenantClaimThatDiffersFromDatabase(t *testing.T) {
	user := validSessionUser()
	claims := security.AccessClaims{
		Subject: user.ID, SessionID: "session-1", EstablishmentID: "018f47d2-e886-7f1c-8e8d-8b38eaf16499", Role: user.Role,
		CredentialVersion: user.CredentialVersion, ExpiresAt: testNow.Add(time.Minute).Unix(),
	}
	middleware := testMiddleware(t, user, claims)
	status := runMiddlewareRequest(t, middleware, http.MethodGet, "/api/v1/auth/me", []*http.Cookie{{Name: accessCookieName, Value: "access"}}, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestMiddlewareRestrictsTemporaryPasswordFromEstablishment(t *testing.T) {
	user := validSessionUser()
	user.PasswordChangeRequired = true
	claims := security.AccessClaims{
		Subject: user.ID, SessionID: "session-1", EstablishmentID: user.EstablishmentID, Role: user.Role,
		CredentialVersion: user.CredentialVersion, ExpiresAt: testNow.Add(time.Minute).Unix(),
	}
	middleware := testMiddleware(t, user, claims)
	status := runMiddlewareRequest(t, middleware, http.MethodGet, "/api/v1/establishment", []*http.Cookie{{Name: accessCookieName, Value: "access"}}, nil)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func testMiddleware(t *testing.T, user User, claims security.AccessClaims) *Middleware {
	t.Helper()
	signer, err := security.NewHMACSigner(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatalf("new CSRF signer: %v", err)
	}
	middleware := NewMiddleware(userStoreStub{user: user}, accessVerifierStub{claims: claims}, signer, []string{"https://localhost:8443"})
	middleware.now = func() time.Time { return testNow }
	return middleware
}

func runMiddlewareRequest(t *testing.T, middleware *Middleware, method, path string, cookies []*http.Cookie, headers map[string]string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("request_id", "request-123") }, middleware.Handle())
	router.Handle(method, path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(method, path, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder.Code
}
