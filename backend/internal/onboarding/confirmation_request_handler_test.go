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
	"barbershop-app/backend/internal/platform/ratelimit"

	"github.com/gin-gonic/gin"
)

func TestEmailConfirmationRequestHandlerReturnsAcceptedWithoutAccountState(t *testing.T) {
	requester := &emailConfirmationRequesterStub{}
	recorder := performEmailConfirmationRequest(t, NewHandler(nil, requester), `{"email":"Owner@Example.COM"}`)

	if recorder.Code != http.StatusAccepted || recorder.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
	if requester.input.Email != "Owner@Example.COM" || requester.input.IP != "192.0.2.20" {
		t.Fatalf("input = %#v", requester.input)
	}
}

func TestEmailConfirmationRequestHandlerRejectsInvalidEmail(t *testing.T) {
	recorder := performEmailConfirmationRequest(t, NewHandler(nil, &emailConfirmationRequesterStub{err: auth.ErrInvalidEmail}), `{"email":"owner@example.com"}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"field":"email"`) || !strings.Contains(recorder.Body.String(), `"code":"invalid_email"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestEmailConfirmationRequestHandlerReturnsRetryAfter(t *testing.T) {
	requester := &emailConfirmationRequesterStub{err: ratelimit.Error{RetryAfter: 90 * time.Second}}
	recorder := performEmailConfirmationRequest(t, NewHandler(nil, requester), `{"email":"owner@example.com"}`)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "90" {
		t.Fatalf("status = %d, retry-after = %q, body = %s", recorder.Code, recorder.Header().Get("Retry-After"), recorder.Body.String())
	}
}

func TestEmailConfirmationRequestHandlerLogsDeliveryFailureWithoutSensitiveData(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	requester := &emailConfirmationRequesterStub{result: EmailConfirmationRequestResult{DeliveryFailed: true}}

	recorder := performEmailConfirmationRequest(t, NewHandler(nil, requester), `{"email":"owner@example.com"}`)
	if recorder.Code != http.StatusAccepted || !strings.Contains(logs.String(), "email_confirmation_delivery_failed") || !strings.Contains(logs.String(), "request-123") {
		t.Fatalf("status = %d, log = %q", recorder.Code, logs.String())
	}
	if strings.Contains(logs.String(), "owner@example.com") {
		t.Fatalf("log contains recipient: %s", logs.String())
	}
}

func TestEmailConfirmationRequestHandlerDoesNotExposeSensitiveInternalError(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	requester := &emailConfirmationRequesterStub{err: errors.New("owner@example.com token-secret")}

	recorder := performEmailConfirmationRequest(t, NewHandler(nil, requester), `{"email":"owner@example.com"}`)
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	for _, output := range []string{recorder.Body.String(), logs.String()} {
		if strings.Contains(output, "owner@example.com") || strings.Contains(output, "token-secret") {
			t.Fatalf("output contains sensitive data: %s", output)
		}
	}
}

func TestEmailConfirmationRequestHandlerRejectsUnknownFields(t *testing.T) {
	recorder := performEmailConfirmationRequest(t, NewHandler(nil, &emailConfirmationRequesterStub{}), `{"email":"owner@example.com","token":"secret"}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func performEmailConfirmationRequest(t *testing.T, handler *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-confirmation", strings.NewReader(body))
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Request.RemoteAddr = "192.0.2.20:1234"
	ginContext.Set("request_id", "request-123")
	handler.PostAuthEmailConfirmation(ginContext)
	return recorder
}

type emailConfirmationRequesterStub struct {
	input  EmailConfirmationRequestInput
	result EmailConfirmationRequestResult
	err    error
}

func (stub *emailConfirmationRequesterStub) Request(_ context.Context, input EmailConfirmationRequestInput) (EmailConfirmationRequestResult, error) {
	stub.input = input
	return stub.result, stub.err
}
