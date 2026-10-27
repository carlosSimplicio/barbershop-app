//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPatchEstablishmentSettingsUpdatesSubmittedValuesAndPreservesOmittedValues(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())

	response := application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"name": "  Barbearia Atualizada  ", "description": "  Primeira descrição  ", "minimum_booking_notice_minutes": 20,
		"booking_horizon_days": 60, "customer_change_deadline_minutes": 120, "contact_phone": "(11) 99999-8888",
		"social_links": []map[string]string{
			{"label": "Instagram", "url": "https://instagram.com/central"},
			{"label": "Site", "url": "https://barbearia.example.test"},
		},
	})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"address": "Rua das Flores, 10",
	})
	assertStatus(t, response, http.StatusNoContent)

	settings := getEstablishmentSettings(t, application)
	if settings.Name != "Barbearia Atualizada" || settings.Description == nil || *settings.Description != "Primeira descrição" || settings.Address == nil || *settings.Address != "Rua das Flores, 10" ||
		settings.MinimumBookingNoticeMinutes == nil || *settings.MinimumBookingNoticeMinutes != 20 {
		t.Fatalf("PATCH did not preserve omitted settings: %+v", settings)
	}
	if settings.ContactPhone == nil || *settings.ContactPhone != "+5511999998888" || len(settings.SocialLinks) != 2 ||
		settings.SocialLinks[0].Label != "Instagram" || settings.SocialLinks[1].Label != "Site" ||
		settings.BookingHorizonDays == nil || *settings.BookingHorizonDays != 60 || settings.CustomerChangeDeadlineMinutes == nil || *settings.CustomerChangeDeadlineMinutes != 120 {
		t.Fatalf("PATCH did not save normalized phone, ordered links and limits: %+v", settings)
	}

	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"description": nil, "address": nil, "contact_phone": nil, "minimum_booking_notice_minutes": nil,
		"booking_horizon_days": nil, "customer_change_deadline_minutes": nil, "social_links": []map[string]string{},
	})
	assertStatus(t, response, http.StatusNoContent)
	settings = getEstablishmentSettings(t, application)
	if settings.Description != nil || settings.Address != nil || settings.ContactPhone != nil || settings.MinimumBookingNoticeMinutes != nil ||
		settings.BookingHorizonDays != nil || settings.CustomerChangeDeadlineMinutes != nil || len(settings.SocialLinks) != 0 {
		t.Fatalf("PATCH did not clear explicit nulls and empty links: %+v", settings)
	}
}

func TestPatchEstablishmentSettingsValidatesFieldsAndLocksOperationalTimezone(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	for _, test := range []struct {
		body  map[string]any
		field string
	}{
		{body: map[string]any{"name": " "}, field: "name"},
		{body: map[string]any{"time_zone": "Not/A-Timezone"}, field: "time_zone"},
		{body: map[string]any{"description": strings.Repeat("a", 501)}, field: "description"},
		{body: map[string]any{"contact_phone": "abc"}, field: "contact_phone"},
		{body: map[string]any{"social_links": []map[string]string{{"label": "Site", "url": "https://user:secret@example.test"}}}, field: "social_links.0.url"},
		{body: map[string]any{"booking_horizon_days": 0}, field: "booking_horizon_days"},
	} {
		response := application.request(t, http.MethodPatch, "/api/v1/establishment/settings", test.body)
		assertStatus(t, response, http.StatusBadRequest)
		var errorResponse struct {
			Fields []struct {
				Field string `json:"field"`
			} `json:"fields"`
		}
		decodeJSON(t, response, &errorResponse)
		if len(errorResponse.Fields) != 1 || errorResponse.Fields[0].Field != test.field {
			t.Fatalf("validation fields = %#v, want %q", errorResponse.Fields, test.field)
		}
	}
	response := application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"slug": "altered-slug", "operational_data_started_at": "2026-10-27T12:00:00Z",
	})
	assertStatus(t, response, http.StatusBadRequest)
	if settings := getEstablishmentSettings(t, application); settings.Slug != defaultOwnerRegistration().Slug {
		t.Fatalf("PATCH changed immutable slug: %q", settings.Slug)
	}

	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"time_zone": "America/Recife"})
	assertStatus(t, response, http.StatusNoContent)
	establishmentID := authenticatedEstablishmentID(t, application)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishments SET operational_data_started_at = $2 WHERE id = $1`, establishmentID, time.Now().UTC()); err != nil {
		t.Fatalf("lock operational time zone fixture: %v", err)
	}
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"time_zone": "America/Recife"})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"time_zone": "America/Manaus"})
	assertStatus(t, response, http.StatusConflict)
	var lockedResponse struct {
		Fields []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"fields"`
	}
	decodeJSON(t, response, &lockedResponse)
	if len(lockedResponse.Fields) != 1 || lockedResponse.Fields[0].Field != "time_zone" || lockedResponse.Fields[0].Code != "time_zone_locked" {
		t.Fatalf("timezone lock response = %#v", lockedResponse)
	}
	if settings := getEstablishmentSettings(t, application); settings.TimeZone != "America/Recife" {
		t.Fatalf("time zone changed despite operational data: %q", settings.TimeZone)
	}
}

func TestPatchEstablishmentSettingsRequiresCsrfWriteAccessAndOwnerTenant(t *testing.T) {
	application := newTestApplication(t)
	firstOwner := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, firstOwner)
	firstID := authenticatedEstablishmentID(t, application)
	response := requestWithCSRF(t, application, map[string]any{"description": "primeiro tenant"}, "invalid-token")
	assertStatus(t, response, http.StatusForbidden)

	secondOwner := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, secondOwner)
	secondID := authenticatedEstablishmentID(t, application)
	if firstID == secondID {
		t.Fatal("test owners unexpectedly share a tenant")
	}
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"description": "segundo tenant"})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": firstOwner.Email, "password": firstOwner.Password})
	assertStatus(t, response, http.StatusOK)
	if settings := getEstablishmentSettings(t, application); settings.Description != nil {
		t.Fatalf("second tenant patch changed first tenant: %+v", settings)
	}

	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2 WHERE establishment_id = $3`,
		time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour), firstID); err != nil {
		t.Fatalf("expire first tenant trial fixture: %v", err)
	}
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"description": "não permitido"})
	assertStatus(t, response, http.StatusForbidden)
	var readOnlyError struct {
		Code string `json:"code"`
	}
	decodeJSON(t, response, &readOnlyError)
	if readOnlyError.Code != "read_only" {
		t.Fatalf("expired trial error code = %q", readOnlyError.Code)
	}
	if settings := getEstablishmentSettings(t, application); settings.Description != nil {
		t.Fatalf("read-only patch changed establishment settings: %+v", settings)
	}
}

func TestPatchEstablishmentSettingsRejectsProfessional(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET password_change_required = false WHERE normalized_email = 'ana@example.test'`); err != nil {
		t.Fatalf("complete professional credentials fixture: %v", err)
	}
	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"description": "não permitido"})
	assertStatus(t, response, http.StatusForbidden)
	var forbiddenError struct {
		Code string `json:"code"`
	}
	decodeJSON(t, response, &forbiddenError)
	if forbiddenError.Code != "forbidden" {
		t.Fatalf("professional settings error code = %q", forbiddenError.Code)
	}
}

func TestPatchEstablishmentSettingsRollsBackAllFieldsWhenLinkWriteFails(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	response := application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"description": "Descrição anterior", "social_links": []map[string]string{{"label": "Site", "url": "https://example.test"}},
	})
	assertStatus(t, response, http.StatusNoContent)
	if _, err := integrationPool.Exec(context.Background(), `
		CREATE FUNCTION reject_test_establishment_link() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.label = 'Falhar' THEN RAISE EXCEPTION 'test-only link write failure'; END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER reject_test_establishment_link BEFORE INSERT ON establishment_social_links
		FOR EACH ROW EXECUTE FUNCTION reject_test_establishment_link()`); err != nil {
		t.Fatalf("create link failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = integrationPool.Exec(context.Background(), `DROP TRIGGER IF EXISTS reject_test_establishment_link ON establishment_social_links; DROP FUNCTION IF EXISTS reject_test_establishment_link()`)
	})

	response = application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"description": "Não pode persistir", "social_links": []map[string]string{{"label": "Falhar", "url": "https://example.test/falhar"}},
	})
	assertStatus(t, response, http.StatusInternalServerError)
	var errorResponse struct {
		Code string `json:"code"`
	}
	decodeJSON(t, response, &errorResponse)
	if errorResponse.Code != "internal_error" {
		t.Fatalf("unexpected link write error: %+v", errorResponse)
	}
	settings := getEstablishmentSettings(t, application)
	if settings.Description == nil || *settings.Description != "Descrição anterior" || len(settings.SocialLinks) != 1 || settings.SocialLinks[0].Label != "Site" {
		t.Fatalf("failed settings transaction left partial changes: %+v", settings)
	}
}

func requestWithCSRF(t *testing.T, application *testApplication, body any, csrf string) *http.Response {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPatch, application.server.URL+"/api/v1/establishment/settings", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("X-CSRF-Token", csrf)
	response, err := application.client.Do(request)
	if err != nil {
		t.Fatalf("perform PATCH with invalid CSRF token: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}
