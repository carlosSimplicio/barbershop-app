package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLoginHandlerReturnsPrincipalAndSecureCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &loginStoreStub{user: User{
		ID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: "018f47d2-e886-7f1c-8e8d-8b38eaf16464",
		Email: "user@example.com", NormalizedEmail: "user@example.com", Role: "professional", Active: true,
		EmailConfirmed: true, CredentialVersion: 1, PasswordHash: "stored-hash", PasswordChangeRequired: true,
	}, attempts: map[string][]time.Time{}}
	service := testLoginService(store, true)
	handler := NewHandler(service, nil, nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"senha-segura"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("request_id", "request-123")

	handler.PostAuthLogin(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"password_change_required":true`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("cookie %s does not have required protections: %#v", cookie.Name, cookie)
		}
	}
}

func TestLoginHandlerRejectsUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(testLoginService(&loginStoreStub{attempts: map[string][]time.Time{}}, false), nil, nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"senha-segura","admin":true}`))
	context.Set("request_id", "request-123")

	handler.PostAuthLogin(context)

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
