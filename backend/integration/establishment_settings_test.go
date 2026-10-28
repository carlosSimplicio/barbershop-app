//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEstablishmentSettingsEndpointAuthorizesOwnerAndIsolatesTenant(t *testing.T) {
	application := newTestApplication(t)
	response := application.request(t, http.MethodGet, "/api/v1/establishment/settings", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	owner := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, owner)
	firstID := authenticatedEstablishmentID(t, application)
	defaultSettings := getEstablishmentSettings(t, application)
	if defaultSettings.Description != nil || defaultSettings.Address != nil || defaultSettings.ContactPhone != nil ||
		defaultSettings.MinimumBookingNoticeMinutes != nil || defaultSettings.BookingHorizonDays != nil ||
		defaultSettings.CustomerChangeDeadlineMinutes != nil || defaultSettings.Logo != nil || len(defaultSettings.SocialLinks) != 0 ||
		!defaultSettings.TimeZoneEditable || !defaultSettings.CanWrite || defaultSettings.IsComplete || defaultSettings.IsPublishable {
		t.Fatalf("unexpected settings before profile completion: %+v", defaultSettings)
	}
	logoHash := sha256.Sum256([]byte("fixture-logo"))
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishments SET description = 'Apresentação', address = 'Rua Um, 10', contact_phone = '+5511999999999', minimum_booking_notice_minutes = 30, booking_horizon_days = 45, customer_change_deadline_minutes = 120, operational_data_started_at = $2 WHERE id = $1`, firstID, time.Now().UTC()); err != nil {
		t.Fatalf("persist establishment settings fixture: %v", err)
	}
	if _, err := integrationPool.Exec(context.Background(), `INSERT INTO establishment_social_links (establishment_id, position, label, url) VALUES ($1, 0, 'Instagram', 'https://instagram.com/central')`, firstID); err != nil {
		t.Fatalf("persist social link fixture: %v", err)
	}
	if _, err := integrationPool.Exec(context.Background(), `WITH logo AS (INSERT INTO establishment_logos (establishment_id, object_key, content_type, byte_size, width, height, content_sha256) VALUES ($1, 'internal-object-key-secret', 'image/png', 128, 64, 32, $2) RETURNING id) UPDATE establishments SET logo_id = logo.id FROM logo WHERE establishments.id = $1`, firstID, logoHash[:]); err != nil {
		t.Fatalf("persist logo metadata fixture: %v", err)
	}

	settings, settingsRaw := fetchEstablishmentSettings(t, application)
	if settings.ID != firstID || settings.Name != owner.EstablishmentName || settings.Slug != owner.Slug ||
		settings.Description == nil || *settings.Description != "Apresentação" || settings.Address == nil || *settings.Address != "Rua Um, 10" ||
		settings.ContactPhone == nil || *settings.ContactPhone != "+5511999999999" || settings.MinimumBookingNoticeMinutes == nil || *settings.MinimumBookingNoticeMinutes != 30 ||
		settings.BookingHorizonDays == nil || *settings.BookingHorizonDays != 45 || settings.CustomerChangeDeadlineMinutes == nil || *settings.CustomerChangeDeadlineMinutes != 120 ||
		settings.TimeZoneEditable || !settings.CanWrite || settings.SlugConflict || !settings.IsComplete || !settings.IsPublishable ||
		settings.Logo == nil || settings.Logo.URL == nil || *settings.Logo.URL != "/api/v1/public/establishments/"+owner.Slug+"/logo" || settings.Logo.ContentType != "image/png" ||
		len(settings.SocialLinks) != 1 || settings.SocialLinks[0].Label != "Instagram" || settings.SocialLinks[0].URL != "https://instagram.com/central" {
		t.Fatalf("unexpected establishment settings: %+v", settings)
	}
	if containsAny(settingsRaw, "internal-object-key-secret", "fixture-logo") {
		t.Fatalf("settings response exposes internal logo data: %s", settingsRaw)
	}

	response = application.request(t, http.MethodGet, "/api/v1/establishment", nil)
	assertStatus(t, response, http.StatusOK)
	var basic map[string]any
	decodeJSON(t, response, &basic)
	if _, exists := basic["description"]; exists {
		t.Fatalf("basic establishment response changed contract: %#v", basic)
	}

	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2`,
		time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("expire owner trial: %v", err)
	}
	if expiredSettings := getEstablishmentSettings(t, application); !expiredSettings.IsPublishable || expiredSettings.CanWrite {
		t.Fatalf("expired trial settings should remain readable and read-only: %+v", expiredSettings)
	}

	otherOwner := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, otherOwner)
	secondID := authenticatedEstablishmentID(t, application)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishments SET slug = 'admin' WHERE id = $1`, secondID); err != nil {
		t.Fatalf("create legacy route conflict fixture: %v", err)
	}
	if _, err := integrationPool.Exec(context.Background(), `WITH logo AS (INSERT INTO establishment_logos (establishment_id, object_key, content_type, byte_size, width, height, content_sha256) VALUES ($1, 'other-internal-object-key', 'image/png', 64, 16, 16, $2) RETURNING id) UPDATE establishments SET logo_id = logo.id FROM logo WHERE establishments.id = $1`, secondID, logoHash[:]); err != nil {
		t.Fatalf("persist logo for conflicting legacy slug: %v", err)
	}
	otherSettings := getEstablishmentSettings(t, application)
	if secondID == firstID || otherSettings.ID != secondID || otherSettings.Description != nil || otherSettings.Logo == nil || otherSettings.Logo.URL != nil || len(otherSettings.SocialLinks) != 0 ||
		!otherSettings.SlugConflict || otherSettings.IsPublishable || !otherSettings.IsComplete {
		t.Fatalf("second tenant received another establishment's settings: %+v", otherSettings)
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": owner.Email, "password": owner.Password})
	assertStatus(t, response, http.StatusOK)
	if firstSettings := getEstablishmentSettings(t, application); firstSettings.ID != firstID || firstSettings.Logo == nil || firstSettings.Description == nil {
		t.Fatalf("first tenant settings were not restored with its session: %+v", firstSettings)
	}
}

func TestEstablishmentSettingsEndpointRejectsProfessional(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET password_change_required = false WHERE normalized_email = 'ana@example.test'`); err != nil {
		t.Fatalf("complete professional credentials fixture: %v", err)
	}
	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodGet, "/api/v1/establishment/settings", nil)
	assertStatus(t, response, http.StatusForbidden)
}

type establishmentSettingsResponse struct {
	ID                            string                     `json:"id"`
	Name                          string                     `json:"name"`
	Slug                          string                     `json:"slug"`
	TimeZone                      string                     `json:"time_zone"`
	Description                   *string                    `json:"description"`
	Address                       *string                    `json:"address"`
	ContactPhone                  *string                    `json:"contact_phone"`
	SocialLinks                   []establishmentSocialLink  `json:"social_links"`
	MinimumBookingNoticeMinutes   *int32                     `json:"minimum_booking_notice_minutes"`
	BookingHorizonDays            *int32                     `json:"booking_horizon_days"`
	CustomerChangeDeadlineMinutes *int32                     `json:"customer_change_deadline_minutes"`
	Logo                          *establishmentLogoMetadata `json:"logo"`
	TimeZoneEditable              bool                       `json:"time_zone_editable"`
	CanWrite                      bool                       `json:"can_write"`
	SlugConflict                  bool                       `json:"slug_conflict"`
	IsComplete                    bool                       `json:"is_complete"`
	IsPublishable                 bool                       `json:"is_publishable"`
}

type establishmentSocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type establishmentLogoMetadata struct {
	URL         *string `json:"url"`
	ContentType string  `json:"content_type"`
}

func getEstablishmentSettings(t *testing.T, application *testApplication) establishmentSettingsResponse {
	t.Helper()
	settings, _ := fetchEstablishmentSettings(t, application)
	return settings
}

func fetchEstablishmentSettings(t *testing.T, application *testApplication) (establishmentSettingsResponse, string) {
	t.Helper()
	body, err := io.ReadAll(settingsResponse(t, application).Body)
	if err != nil {
		t.Fatalf("read settings response: %v", err)
	}
	var settings establishmentSettingsResponse
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatalf("decode settings response: %v", err)
	}
	return settings, string(body)
}

func settingsResponse(t *testing.T, application *testApplication) *http.Response {
	t.Helper()
	response := application.request(t, http.MethodGet, "/api/v1/establishment/settings", nil)
	assertStatus(t, response, http.StatusOK)
	return response
}

func authenticatedEstablishmentID(t *testing.T, application *testApplication) string {
	t.Helper()
	response := application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var principal struct {
		EstablishmentID string `json:"establishment_id"`
	}
	decodeJSON(t, response, &principal)
	return principal.EstablishmentID
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
