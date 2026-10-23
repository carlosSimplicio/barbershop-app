package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type recoveryVerifierStub struct {
	token    string
	password string
	err      error
}

func (stub *recoveryVerifierStub) Verify(_ context.Context, token, password string) error {
	stub.token, stub.password = token, password
	return stub.err
}

func TestPasswordRecoveryVerificationHandlerReturnsNoSession(t *testing.T) {
	verifier := &recoveryVerifierStub{}
	recorder := performRecoveryVerification(t, verifier, `{"token":"opaque-token","new_password":"nova-senha-segura"}`)
	if recorder.Code != http.StatusNoContent || len(recorder.Result().Cookies()) != 0 || verifier.token != "opaque-token" || verifier.password != "nova-senha-segura" {
		t.Fatalf("status = %d, cookies = %#v, verifier = %+v", recorder.Code, recorder.Result().Cookies(), verifier)
	}
}

func TestPasswordRecoveryVerificationHandlerDoesNotLogToken(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	recorder := performRecoveryVerification(t, &recoveryVerifierStub{err: errors.New("token-secret")}, `{"token":"token-secret","new_password":"nova-senha-segura"}`)
	if recorder.Code != http.StatusInternalServerError || strings.Contains(logs.String(), "token-secret") || strings.Contains(recorder.Body.String(), "token-secret") {
		t.Fatalf("status = %d, log = %q, body = %s", recorder.Code, logs.String(), recorder.Body.String())
	}
}

func TestPasswordRecoveryVerificationHandlerRejectsUnknownFields(t *testing.T) {
	recorder := performRecoveryVerification(t, &recoveryVerifierStub{}, `{"token":"value","new_password":"nova-senha-segura","email":"owner@example.test"}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"invalid_request"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func performRecoveryVerification(t *testing.T, verifier PasswordRecoveryVerifier, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, recoveryVerifyPathForTest, strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("request_id", "request-123")
	NewHandler(nil, nil, nil, verifier).PostAuthPasswordRecoveryVerify(context)
	return recorder
}

const recoveryVerifyPathForTest = "/api/v1/auth/password/recovery/verify"
