package establishment

import (
	"context"
	"errors"
	"io"
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
	recorder := performEstablishmentRequest(t, NewHandler(finder, nil, nil, nil), &auth.Principal{
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
	}}, nil, nil, nil), &auth.Principal{EstablishmentID: testEstablishmentID}, "")

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"trial":null`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerRejectsRequestWithoutAuthenticatedPrincipal(t *testing.T) {
	recorder := performEstablishmentRequest(t, NewHandler(&finderStub{}, nil, nil, nil), nil, "")

	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), `"code":"authentication_required"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerDoesNotExposeInternalFailure(t *testing.T) {
	recorder := performEstablishmentRequest(t, NewHandler(&finderStub{err: errors.New("database-secret")}, nil, nil, nil), &auth.Principal{
		EstablishmentID: testEstablishmentID,
	}, "")

	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "database-secret") {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestPublicLogoReadFailureReturnsInternalErrorBeforeWritingImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/public/establishments/example/logo", nil)
	ginContext.Params = gin.Params{{Key: "slug", Value: "example"}}
	ginContext.Set("request_id", "request-123")

	handler := NewHandler(nil, nil, nil, failingPublicLogoReader{})
	handler.GetPublicEstablishmentLogo(ginContext, "example")

	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "partial") {
		t.Fatalf("status = %d, body = %q; want generic 500 without partial image bytes", recorder.Code, recorder.Body.String())
	}
}

func TestPublicLogoNotModifiedClosesStorageWithoutReadingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/public/establishments/example/logo", nil)
	ginContext.Request.Header.Set("If-None-Match", `"etag"`)
	ginContext.Params = gin.Params{{Key: "slug", Value: "example"}}
	body := &trackedPublicLogoBody{}
	reader := trackedPublicLogoReader{body: body}
	handler := NewHandler(nil, nil, nil, reader)

	handler.GetPublicEstablishmentLogo(ginContext, "example")

	if ginContext.Writer.Status() != http.StatusNotModified || recorder.Body.Len() != 0 {
		t.Fatalf("status/body = %d/%q; want empty 304", ginContext.Writer.Status(), recorder.Body.String())
	}
	if body.reads != 0 || body.closed != 1 {
		t.Fatalf("storage reads/closes = %d/%d; want 0/1", body.reads, body.closed)
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

type failingPublicLogoReader struct{}

func (failingPublicLogoReader) OpenPublicLogo(context.Context, string) (PublicLogo, io.ReadCloser, error) {
	return PublicLogo{ContentType: "image/png", ByteSize: 7, ETag: `"etag"`}, failingPublicLogoBody{}, nil
}

type failingPublicLogoBody struct{}

func (failingPublicLogoBody) Read(buffer []byte) (int, error) {
	return copy(buffer, "partial"), errors.New("storage read failed")
}

func (failingPublicLogoBody) Close() error { return nil }

type trackedPublicLogoReader struct{ body *trackedPublicLogoBody }

func (reader trackedPublicLogoReader) OpenPublicLogo(context.Context, string) (PublicLogo, io.ReadCloser, error) {
	return PublicLogo{ContentType: "image/png", ByteSize: 4, ETag: `"etag"`}, reader.body, nil
}

type trackedPublicLogoBody struct {
	reads  int
	closed int
}

func (body *trackedPublicLogoBody) Read([]byte) (int, error) {
	body.reads++
	return 0, errors.New("body should not be read")
}

func (body *trackedPublicLogoBody) Close() error {
	body.closed++
	return nil
}
