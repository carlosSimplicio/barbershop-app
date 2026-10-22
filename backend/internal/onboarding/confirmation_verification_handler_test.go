package onboarding

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"barbershop-app/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestEmailConfirmationVerificationHandlerReturnsPrincipalAndSecureCookies(t *testing.T) {
	verifier := &emailConfirmationVerifierStub{result: auth.LoginResult{
		Principal: auth.Principal{
			UserID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: "018f47d2-e886-7f1c-8e8d-8b38eaf16464",
			Email: "owner@example.com", Role: "owner",
		},
		AccessToken: "access", CSRFToken: "csrf",
	}}
	recorder := performEmailConfirmationVerification(t, NewHandler(nil, nil, verifier), `{"token":"confirmation-token"}`)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"role":"owner"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if verifier.token != "confirmation-token" || len(recorder.Result().Cookies()) != 2 {
		t.Fatalf("token = %q, cookies = %#v", verifier.token, recorder.Result().Cookies())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("insecure cookie: %#v", cookie)
		}
	}
}

func TestEmailConfirmationVerificationHandlerUsesUniformInvalidError(t *testing.T) {
	recorder := performEmailConfirmationVerification(t, NewHandler(nil, nil, &emailConfirmationVerifierStub{err: ErrInvalidEmailConfirmation}), `{"token":"used-or-expired"}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_email_confirmation"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestEmailConfirmationVerificationHandlerDoesNotLogToken(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	recorder := performEmailConfirmationVerification(t, NewHandler(nil, nil, &emailConfirmationVerifierStub{err: errors.New("token-secret")}), `{"token":"token-secret"}`)
	if recorder.Code != http.StatusInternalServerError || strings.Contains(logs.String(), "token-secret") || strings.Contains(recorder.Body.String(), "token-secret") {
		t.Fatalf("status = %d, log = %q, body = %s", recorder.Code, logs.String(), recorder.Body.String())
	}
}

func TestEmailConfirmationVerificationHandlerRejectsUnknownFields(t *testing.T) {
	recorder := performEmailConfirmationVerification(t, NewHandler(nil, nil, &emailConfirmationVerifierStub{}), `{"token":"value","email":"owner@example.com"}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func performEmailConfirmationVerification(t *testing.T, handler *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-confirmation/verify", strings.NewReader(body))
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set("request_id", "request-123")
	handler.PostAuthEmailConfirmationVerify(ginContext)
	return recorder
}

type emailConfirmationVerifierStub struct {
	token  string
	result auth.LoginResult
	err    error
}

func (stub *emailConfirmationVerifierStub) Verify(_ context.Context, token string) (auth.LoginResult, error) {
	stub.token = token
	return stub.result, stub.err
}
