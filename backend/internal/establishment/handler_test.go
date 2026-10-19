package establishment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestHandlerReturnsOnlyAuthenticatedEstablishment(t *testing.T) {
	finder := &finderStub{establishment: Establishment{
		ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
		Trial: &Trial{
			Status:   TrialStatusActive,
			StartsAt: time.Date(2026, time.October, 19, 14, 30, 0, 0, time.UTC),
			EndsAt:   time.Date(2026, time.November, 2, 14, 30, 0, 0, time.UTC),
		},
	}}
	recorder := performEstablishmentRequest(t, NewHandler(finder), &auth.Principal{
		UserID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: testEstablishmentID, Role: "owner",
	}, "?establishment_id="+otherEstablishmentID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if finder.principal.EstablishmentID != testEstablishmentID {
		t.Fatalf("queried establishment = %q", finder.principal.EstablishmentID)
	}
	wantFragments := []string{
		`"id":"` + testEstablishmentID + `"`, `"name":"Barbearia Central"`, `"slug":"barbearia-central"`,
		`"time_zone":"America/Sao_Paulo"`, `"status":"active"`, `"starts_at":"2026-10-19T14:30:00Z"`,
		`"ends_at":"2026-11-02T14:30:00Z"`,
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(recorder.Body.String(), fragment) {
			t.Errorf("body does not contain %s: %s", fragment, recorder.Body.String())
		}
	}
	if strings.Contains(recorder.Body.String(), otherEstablishmentID) {
		t.Fatalf("body exposes external establishment: %s", recorder.Body.String())
	}
}

func TestHandlerReturnsNullTrialBeforeConfirmation(t *testing.T) {
	recorder := performEstablishmentRequest(t, NewHandler(&finderStub{establishment: Establishment{
		ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
	}}), &auth.Principal{EstablishmentID: testEstablishmentID}, "")

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"trial":null`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerRejectsRequestWithoutAuthenticatedPrincipal(t *testing.T) {
	recorder := performEstablishmentRequest(t, NewHandler(&finderStub{}), nil, "")

	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), `"code":"authentication_required"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerDoesNotExposeInternalFailure(t *testing.T) {
	recorder := performEstablishmentRequest(t, NewHandler(&finderStub{err: errors.New("database-secret")}), &auth.Principal{
		EstablishmentID: testEstablishmentID,
	}, "")

	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "database-secret") {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func performEstablishmentRequest(t *testing.T, handler *Handler, principal *auth.Principal, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/establishment"+rawQuery, nil)
	ginContext.Set("request_id", "request-123")
	if principal != nil {
		ginContext.Set("auth_principal", *principal)
	}
	handler.GetEstablishment(ginContext)
	return recorder
}

type finderStub struct {
	principal     auth.Principal
	establishment Establishment
	err           error
}

func (stub *finderStub) Get(_ context.Context, principal auth.Principal) (Establishment, error) {
	stub.principal = principal
	return stub.establishment, stub.err
}
