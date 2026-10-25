//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

type professionalPage struct {
	Items []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Active bool   `json:"active"`
	} `json:"items"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

func TestProfessionalListPaginatesAndIsolatesEstablishments(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	for _, name := range []string{"Ana", "Bia", "Caio"} {
		createProfessionalForList(t, application, name, name+"@example.test")
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE professionals AS p
		SET created_at = CASE u.normalized_email
			WHEN 'ana@example.test' THEN '2026-10-22T12:00:00Z'::timestamptz
			WHEN 'bia@example.test' THEN '2026-10-22T12:01:00Z'::timestamptz
			ELSE '2026-10-22T12:02:00Z'::timestamptz END
		FROM users AS u WHERE u.id = p.user_id`); err != nil {
		t.Fatalf("set deterministic creation order: %v", err)
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET active = false WHERE normalized_email = 'bia@example.test'`); err != nil {
		t.Fatalf("deactivate list fixture: %v", err)
	}

	first := getProfessionalPage(t, application, "/api/v1/professionals?limit=2&offset=0")
	if first.Limit != 2 || first.Offset != 0 || !first.HasMore || len(first.Items) != 2 ||
		first.Items[0].Name != "Caio" || first.Items[1].Name != "Bia" || first.Items[1].Active {
		t.Fatalf("first page = %#v", first)
	}
	second := getProfessionalPage(t, application, "/api/v1/professionals?limit=2&offset=2")
	if second.Limit != 2 || second.Offset != 2 || second.HasMore || len(second.Items) != 1 || second.Items[0].Name != "Ana" {
		t.Fatalf("second page = %#v", second)
	}

	other := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, other)
	createProfessionalForList(t, application, "Dora", "dora@example.test")
	otherPage := getProfessionalPage(t, application, "/api/v1/professionals")
	if len(otherPage.Items) != 1 || otherPage.Items[0].Name != "Dora" {
		t.Fatalf("other establishment list = %#v", otherPage)
	}
	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "owner@example.test", "password": "senha-segura",
	})
	assertStatus(t, response, http.StatusOK)
	firstOwnerPage := getProfessionalPage(t, application, "/api/v1/professionals")
	if len(firstOwnerPage.Items) != 3 || firstOwnerPage.Items[0].Name != "Caio" {
		t.Fatalf("first establishment list after switching session = %#v", firstOwnerPage)
	}
}

func TestProfessionalListRemainsReadableAfterTrialExpires(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2`,
		time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("expire trial: %v", err)
	}
	page := getProfessionalPage(t, application, "/api/v1/professionals")
	if len(page.Items) != 1 || page.Items[0].Name != "Ana" {
		t.Fatalf("expired trial list = %#v", page)
	}
}

func TestProfessionalListRequiresOwnerSessionAndValidPagination(t *testing.T) {
	application := newTestApplication(t)
	response := application.request(t, http.MethodGet, "/api/v1/professionals", nil)
	assertStatus(t, response, http.StatusUnauthorized)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	for _, test := range []struct {
		query string
		field string
	}{
		{query: "limit=0", field: "limit"},
		{query: "limit=101", field: "limit"},
		{query: "offset=-1", field: "offset"},
		{query: "limit=invalid", field: "query"},
	} {
		response = application.request(t, http.MethodGet, "/api/v1/professionals?"+test.query, nil)
		assertStatus(t, response, http.StatusBadRequest)
		assertProfessionalFieldError(t, response, test.field)
	}
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "ana@example.test", "password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusOK)
	response = application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodGet, "/api/v1/professionals", nil)
	assertStatus(t, response, http.StatusForbidden)
}

func createProfessionalForList(t *testing.T, application *testApplication, name, email string) {
	t.Helper()
	response := application.request(t, http.MethodPost, "/api/v1/professionals", map[string]string{
		"name": name, "email": email, "temporary_password": "senha-temporaria",
	})
	assertStatus(t, response, http.StatusCreated)
}

func getProfessionalPage(t *testing.T, application *testApplication, path string) professionalPage {
	t.Helper()
	response := application.request(t, http.MethodGet, path, nil)
	assertStatus(t, response, http.StatusOK)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read professionals: %v", err)
	}
	if bytes.Contains(body, []byte("temporary_password")) || bytes.Contains(body, []byte("password_hash")) ||
		bytes.Contains(body, []byte("senha-temporaria")) {
		t.Fatal("professional list exposed credential material")
	}
	var result professionalPage
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode professionals: %v", err)
	}
	if result.Items == nil {
		t.Fatal("professional list returned null items")
	}
	return result
}
