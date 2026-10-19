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
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/ratelimit"

	"github.com/gin-gonic/gin"
)

func TestRegistrationHandlerReturnsAcceptedWithoutSensitiveResponse(t *testing.T) {
	registrar := &registrationRegistrarStub{}
	recorder := performRegistrationRequest(t, NewHandler(registrar), validRegistrationJSON())

	if recorder.Code != http.StatusAccepted || recorder.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
	if registrar.input.Email != "owner@example.com" || registrar.input.IP != "192.0.2.10" {
		t.Fatalf("input = %#v", registrar.input)
	}
}

func TestRegistrationHandlerMapsValidationAndConflictErrorsToFields(t *testing.T) {
	tests := []struct {
		err    error
		status int
		field  string
		code   string
	}{
		{ErrInvalidName, http.StatusBadRequest, "establishment_name", "invalid_name"},
		{ErrInvalidSlug, http.StatusBadRequest, "slug", "invalid_slug"},
		{ErrReservedSlug, http.StatusBadRequest, "slug", "reserved_slug"},
		{ErrInvalidTimeZone, http.StatusBadRequest, "time_zone", "invalid_time_zone"},
		{auth.ErrInvalidEmail, http.StatusBadRequest, "email", "invalid_email"},
		{security.ErrInvalidPassword, http.StatusBadRequest, "password", "invalid_password"},
		{ErrEmailUnavailable, http.StatusConflict, "email", "email_unavailable"},
		{ErrSlugUnavailable, http.StatusConflict, "slug", "slug_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			recorder := performRegistrationRequest(t, NewHandler(&registrationRegistrarStub{err: test.err}), validRegistrationJSON())
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"field":"`+test.field+`"`) || !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestRegistrationHandlerReturnsRetryAfterWhenRateLimited(t *testing.T) {
	recorder := performRegistrationRequest(t, NewHandler(&registrationRegistrarStub{err: ratelimit.Error{RetryAfter: 90 * time.Second}}), validRegistrationJSON())
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "90" {
		t.Fatalf("status = %d, retry-after = %q, body = %s", recorder.Code, recorder.Header().Get("Retry-After"), recorder.Body.String())
	}
}

func TestRegistrationHandlerLogsOnlyOperationalEventWhenDeliveryFails(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	recorder := performRegistrationRequest(t, NewHandler(&registrationRegistrarStub{result: RegistrationResult{DeliveryFailed: true}}), validRegistrationJSON())
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	output := logs.String()
	if !strings.Contains(output, "registration_confirmation_delivery_failed") || !strings.Contains(output, "request-123") {
		t.Fatalf("log = %q", output)
	}
	for _, secret := range []string{"owner@example.com", "senha-segura", "barbearia-central"} {
		if strings.Contains(output, secret) {
			t.Fatalf("log contains sensitive registration value %q: %s", secret, output)
		}
	}
}

func TestRegistrationHandlerDoesNotExposeSensitiveInternalError(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	sensitiveError := errors.New("owner@example.com token-secret senha-segura")

	recorder := performRegistrationRequest(t, NewHandler(&registrationRegistrarStub{err: sensitiveError}), validRegistrationJSON())
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	for _, output := range []string{recorder.Body.String(), logs.String()} {
		for _, secret := range []string{"owner@example.com", "token-secret", "senha-segura"} {
			if strings.Contains(output, secret) {
				t.Fatalf("output contains sensitive value %q: %s", secret, output)
			}
		}
	}
}

func TestRegistrationHandlerRejectsUnknownFields(t *testing.T) {
	body := strings.TrimSuffix(validRegistrationJSON(), "}") + `,"admin":true}`
	recorder := performRegistrationRequest(t, NewHandler(&registrationRegistrarStub{}), body)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func performRegistrationRequest(t *testing.T, handler *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Request.RemoteAddr = "192.0.2.10:1234"
	ginContext.Set("request_id", "request-123")
	handler.PostAuthRegister(ginContext)
	return recorder
}

func validRegistrationJSON() string {
	return `{"establishment_name":"Barbearia Central","slug":"barbearia-central","time_zone":"America/Sao_Paulo","email":"owner@example.com","password":"senha-segura"}`
}

type registrationRegistrarStub struct {
	input  RegistrationInput
	result RegistrationResult
	err    error
}

func (stub *registrationRegistrarStub) Register(_ context.Context, input RegistrationInput) (RegistrationResult, error) {
	stub.input = input
	return stub.result, stub.err
}
