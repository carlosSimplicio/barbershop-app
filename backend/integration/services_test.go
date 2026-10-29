//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

type serviceRecord struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PriceCents      int64  `json:"price_cents"`
	DurationMinutes int64  `json:"duration_minutes"`
	Active          bool   `json:"active"`
}

func createCatalogService(t *testing.T, app *testApplication, name string, price int64) serviceRecord {
	t.Helper()
	r := app.request(t, "POST", "/api/v1/services", map[string]any{"name": name, "price_cents": price})
	assertStatus(t, r, 201)
	var record serviceRecord
	decodeJSON(t, r, &record)
	return record
}
func expireServiceTrial(t *testing.T) {
	t.Helper()
	_, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at=$1,ends_at=$2`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
}
func serviceRawRequest(t *testing.T, app *testApplication, method, path string, body any, origin string, csrf, access bool) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, app.server.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	if csrf {
		cookie := app.cookie(t, csrfCookieName)
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", cookie.Value)
	}
	if access {
		req.AddCookie(app.cookie(t, accessCookieName))
	}
	r, err := (&http.Client{Transport: app.client.Transport}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Body.Close() })
	return r
}
func TestServiceCreationContractAndRefusals(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	first := createCatalogService(t, app, " Corte ", 1)
	if first.Name != "Corte" || first.PriceCents != 1 || first.DurationMinutes != 30 || !first.Active {
		t.Fatalf("record=%+v", first)
	}
	second := createCatalogService(t, app, "Corte", 0)
	if first.ID == second.ID {
		t.Fatal("duplicate names reused identity")
	}
	for _, body := range []map[string]any{{"name": "Corte"}, {"name": "Corte", "price_cents": nil}, {"name": "Corte", "price_cents": 0.5}, {"name": "Corte", "price_cents": -1}, {"name": "Corte", "price_cents": 0, "duration_minutes": 0}, {"name": " ", "price_cents": 0}, {"name": "Corte", "price_cents": 0, "establishment_id": "external"}} {
		assertStatus(t, app.request(t, "POST", "/api/v1/services", body), 400)
	}
	body := map[string]any{"name": "Corte", "price_cents": 9999999, "duration_minutes": 17}
	assertStatus(t, app.request(t, "POST", "/api/v1/services", body), 201)
	assertStatus(t, serviceRawRequest(t, app, "POST", "/api/v1/services", body, trustedOrigin, false, true), 403)
	assertStatus(t, serviceRawRequest(t, app, "POST", "/api/v1/services", body, "https://external.example.test", true, true), 403)
	assertStatus(t, serviceRawRequest(t, app, "POST", "/api/v1/services", body, trustedOrigin, true, false), 401)
	createProfessionalForList(t, app, "Ana", "ana@example.test")
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), 200)
	assertStatus(t, app.request(t, "POST", "/api/v1/services", body), 403)
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), 204)
	assertStatus(t, app.request(t, "POST", "/api/v1/services", body), 403)
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), 200)
	expireServiceTrial(t)
	assertStatus(t, app.request(t, "POST", "/api/v1/services", body), 403)
	var count int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM services`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestServiceCreationUsesOnlyOwnerTenant(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	first := createCatalogService(t, app, "Corte", 1)
	other := defaultOwnerRegistration()
	other.Email = "catalog-other@example.test"
	other.Slug = "catalog-other"
	registerAndConfirmOwner(t, app, other)
	second := createCatalogService(t, app, "Corte", 1)
	var count int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(DISTINCT establishment_id) FROM services WHERE id IN ($1,$2)`, first.ID, second.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("tenants=%d err=%v", count, err)
	}
	response := app.request(t, "POST", "/api/v1/services", map[string]any{"name": "Corte", "price_cents": 0, "duration_minutes": nil})
	assertStatus(t, response, 201)
	var record serviceRecord
	decodeJSON(t, response, &record)
	if record.DurationMinutes != 30 {
		t.Fatalf("null duration=%d", record.DurationMinutes)
	}
}
