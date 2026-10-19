//go:build integration

package integration_test

import (
	"net/http"
	"testing"
)

type ownerRegistration struct {
	EstablishmentName string
	Slug              string
	TimeZone          string
	Email             string
	Password          string
}

func defaultOwnerRegistration() ownerRegistration {
	return ownerRegistration{
		EstablishmentName: "Barbearia Integração",
		Slug:              "barbearia-integracao",
		TimeZone:          "America/Sao_Paulo",
		Email:             "owner@example.test",
		Password:          "senha-segura",
	}
}

func registerOwner(t *testing.T, application *testApplication, registration ownerRegistration) string {
	t.Helper()
	response := application.request(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"establishment_name": registration.EstablishmentName,
		"slug":               registration.Slug,
		"time_zone":          registration.TimeZone,
		"email":              registration.Email,
		"password":           registration.Password,
	})
	assertStatus(t, response, http.StatusAccepted)
	return application.messages.latestToken(t)
}

func registerAndConfirmOwner(t *testing.T, application *testApplication, registration ownerRegistration) {
	t.Helper()
	token := registerOwner(t, application, registration)
	response := application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": token})
	assertStatus(t, response, http.StatusOK)
}
