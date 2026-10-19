//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"
)

func TestAuthenticatedUserReadsOwnEstablishment(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)

	response := application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var principal struct {
		EstablishmentID string `json:"establishment_id"`
	}
	decodeJSON(t, response, &principal)

	response = application.request(t, http.MethodGet, "/api/v1/establishment", nil)
	assertStatus(t, response, http.StatusOK)
	var establishmentResponse struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		TimeZone string `json:"time_zone"`
		Trial    struct {
			Status string    `json:"status"`
			Starts time.Time `json:"starts_at"`
			Ends   time.Time `json:"ends_at"`
		} `json:"trial"`
	}
	decodeJSON(t, response, &establishmentResponse)
	if establishmentResponse.ID != principal.EstablishmentID || establishmentResponse.Name != registration.EstablishmentName ||
		establishmentResponse.Slug != registration.Slug || establishmentResponse.TimeZone != registration.TimeZone ||
		establishmentResponse.Trial.Status != "active" || establishmentResponse.Trial.Ends.Sub(establishmentResponse.Trial.Starts) != 14*24*time.Hour {
		t.Fatalf("unexpected establishment response: %+v", establishmentResponse)
	}
}
