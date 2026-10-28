//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"image/color"
)

func TestGetPublicEstablishmentProfilePublishesOnlyInformationalFields(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())

	client := &http.Client{Transport: application.server.Client().Transport}
	getProfile := func(slug string) *http.Response {
		t.Helper()
		response, err := client.Get(application.server.URL + "/api/v1/public/establishments/" + slug)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}

	incomplete := getProfile(defaultOwnerRegistration().Slug)
	assertStatus(t, incomplete, http.StatusNotFound)
	_ = incomplete.Body.Close()

	response := application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{
		"description": "Cortes e cuidados", "address": "Rua das Flores, 10", "contact_phone": "(11) 99999-8888",
		"social_links":                   []map[string]string{{"label": "Instagram", "url": "https://instagram.com/central"}},
		"minimum_booking_notice_minutes": 30,
	})
	assertStatus(t, response, http.StatusNoContent)
	logo := testLogoPNG(t, color.NRGBA{R: 20, G: 60, B: 100, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", logo, ""), http.StatusNoContent)

	response = getProfile(defaultOwnerRegistration().Slug)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var profile map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		t.Fatalf("decode public establishment profile: %v", err)
	}
	var got struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		LogoURL      string `json:"logo_url"`
		Description  string `json:"description"`
		Address      string `json:"address"`
		ContactPhone string `json:"contact_phone"`
		SocialLinks  []struct {
			Label string `json:"label"`
			URL   string `json:"url"`
		} `json:"social_links"`
	}
	if err := json.Unmarshal(mustJSON(t, profile), &got); err != nil {
		t.Fatalf("decode public profile fields: %v", err)
	}
	if got.Name != defaultOwnerRegistration().EstablishmentName || got.Slug != defaultOwnerRegistration().Slug ||
		got.LogoURL != "/api/v1/public/establishments/"+got.Slug+"/logo" || got.Description != "Cortes e cuidados" ||
		got.Address != "Rua das Flores, 10" || got.ContactPhone != "+5511999998888" || len(got.SocialLinks) != 1 ||
		got.SocialLinks[0].Label != "Instagram" || got.SocialLinks[0].URL != "https://instagram.com/central" {
		t.Fatalf("public profile = %+v", got)
	}
	for _, privateField := range []string{"id", "time_zone", "trial", "minimum_booking_notice_minutes", "can_write", "operational_data_started_at"} {
		if _, exists := profile[privateField]; exists {
			t.Errorf("public profile exposes private field %q", privateField)
		}
	}

	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2 WHERE establishment_id = $3`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour), authenticatedEstablishmentID(t, application)); err != nil {
		t.Fatalf("expire trial fixture: %v", err)
	}
	expiredTrial := getProfile(defaultOwnerRegistration().Slug)
	defer expiredTrial.Body.Close()
	assertStatus(t, expiredTrial, http.StatusOK)

	missing := getProfile("missing-establishment")
	assertStatus(t, missing, http.StatusNotFound)
	_ = missing.Body.Close()

	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishments SET slug = 'api' WHERE id = $1`, authenticatedEstablishmentID(t, application)); err != nil {
		t.Fatalf("create reserved legacy slug fixture: %v", err)
	}
	reserved := getProfile("api")
	assertStatus(t, reserved, http.StatusNotFound)
	_ = reserved.Body.Close()
}

func TestGetPublicEstablishmentProfileOmitsEmptyOptionalFields(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	logo := testLogoPNG(t, color.NRGBA{R: 20, G: 60, B: 100, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", logo, ""), http.StatusNoContent)

	client := &http.Client{Transport: application.server.Client().Transport}
	response, err := client.Get(application.server.URL + "/api/v1/public/establishments/" + defaultOwnerRegistration().Slug)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var profile map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		t.Fatalf("decode public establishment profile: %v", err)
	}
	for _, field := range []string{"description", "address", "contact_phone", "social_links"} {
		if _, exists := profile[field]; exists {
			t.Errorf("empty optional field %q should be omitted", field)
		}
	}
}

func TestGetPublicEstablishmentProfileKeepsProfilesIsolatedBySlug(t *testing.T) {
	application := newTestApplication(t)
	firstOwner := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, firstOwner)
	assertStatus(t, application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"description": "Perfil do primeiro tenant"}), http.StatusNoContent)
	assertStatus(t, uploadLogo(t, application, "first.png", "image/png", testLogoPNG(t, color.NRGBA{R: 20, G: 60, B: 100, A: 255}), ""), http.StatusNoContent)

	secondOwner := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, secondOwner)
	assertStatus(t, application.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]any{"description": "Perfil do segundo tenant"}), http.StatusNoContent)
	assertStatus(t, uploadLogo(t, application, "second.png", "image/png", testLogoPNG(t, color.NRGBA{R: 100, G: 60, B: 20, A: 255}), ""), http.StatusNoContent)

	client := &http.Client{Transport: application.server.Client().Transport}
	for _, expected := range []struct {
		owner       ownerRegistration
		description string
	}{
		{owner: firstOwner, description: "Perfil do primeiro tenant"},
		{owner: secondOwner, description: "Perfil do segundo tenant"},
	} {
		response, err := client.Get(application.server.URL + "/api/v1/public/establishments/" + expected.owner.Slug)
		if err != nil {
			t.Fatal(err)
		}
		var profile struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&profile)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || decodeErr != nil || profile.Name != expected.owner.EstablishmentName ||
			profile.Slug != expected.owner.Slug || profile.Description != expected.description {
			t.Errorf("profile for %q: status=%d profile=%+v decodeErr=%v", expected.owner.Slug, response.StatusCode, profile, decodeErr)
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
