//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
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

type servicePage struct {
	Items    []serviceRecord `json:"items"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
	HasMore  bool            `json:"has_more"`
	CanWrite bool            `json:"can_write"`
}

func getCatalogPage(t *testing.T, app *testApplication, query string) servicePage {
	t.Helper()
	r := app.request(t, "GET", "/api/v1/services"+query, nil)
	assertStatus(t, r, 200)
	var page servicePage
	decodeJSON(t, r, &page)
	return page
}
func TestServiceListPaginationTenantAndExpiredRead(t *testing.T) {
	app := newTestApplication(t)
	assertStatus(t, app.request(t, "GET", "/api/v1/services", nil), 401)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	if page := getCatalogPage(t, app, ""); page.Items == nil || len(page.Items) != 0 || !page.CanWrite {
		t.Fatalf("empty=%+v", page)
	}
	records := []serviceRecord{createCatalogService(t, app, "Corte", 0), createCatalogService(t, app, "Barba", 100), createCatalogService(t, app, "Penteado", 200)}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE services SET created_at='2026-10-20T12:00:00Z', active=false`); err != nil {
		t.Fatal(err)
	}
	page := getCatalogPage(t, app, "?limit=2&offset=0")
	next := getCatalogPage(t, app, "?limit=2&offset=2")
	if len(page.Items) != 2 || !page.HasMore || len(next.Items) != 1 || next.HasMore || page.Items[0].Active || !(page.Items[0].ID > page.Items[1].ID && page.Items[1].ID > next.Items[0].ID) {
		t.Fatalf("page=%+v next=%+v", page, next)
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?offset=-1", "?offset=2147483648", "?limit=invalid"} {
		assertStatus(t, app.request(t, "GET", "/api/v1/services"+query, nil), 400)
	}
	other := defaultOwnerRegistration()
	other.Email = "list-other@example.test"
	other.Slug = "list-other"
	registerAndConfirmOwner(t, app, other)
	own := createCatalogService(t, app, "Outro serviço", 1)
	page = getCatalogPage(t, app, "")
	if len(page.Items) != 1 || page.Items[0].ID != own.ID {
		t.Fatalf("cross-tenant page=%+v old=%+v", page, records)
	}
	createProfessionalForList(t, app, "Ana", "list-prof@example.test")
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "list-prof@example.test", "password": "senha-temporaria"}), 200)
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), 204)
	assertStatus(t, app.request(t, "GET", "/api/v1/services", nil), 403)
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": other.Email, "password": other.Password}), 200)
	expireServiceTrial(t)
	page = getCatalogPage(t, app, "")
	if page.CanWrite || len(page.Items) != 1 {
		t.Fatalf("expired=%+v", page)
	}
	if _, err := integrationPool.Exec(context.Background(), `DELETE FROM establishment_trials`); err != nil {
		t.Fatal(err)
	}
	if getCatalogPage(t, app, "").CanWrite {
		t.Fatal("absent trial can write")
	}
}

func TestServiceIndividualReadHidesOtherTenants(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	record := createCatalogService(t, app, "Corte", 1)
	path := "/api/v1/services/" + record.ID
	response := app.request(t, "GET", path, nil)
	assertStatus(t, response, 200)
	var got serviceRecord
	decodeJSON(t, response, &got)
	if got != record {
		t.Fatalf("got=%+v want=%+v", got, record)
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE services SET active=false WHERE id=$1`, record.ID); err != nil {
		t.Fatal(err)
	}
	expireServiceTrial(t)
	response = app.request(t, "GET", path, nil)
	assertStatus(t, response, 200)
	decodeJSON(t, response, &got)
	if got.Active {
		t.Fatal("inactive not returned")
	}
	assertStatus(t, app.request(t, "GET", "/api/v1/services/invalid", nil), 400)
	assertStatus(t, app.request(t, "GET", "/api/v1/services/771d0b88-237b-4f35-89ad-850ffbd894d3", nil), 404)
	other := defaultOwnerRegistration()
	other.Email = "get-other@example.test"
	other.Slug = "get-other"
	registerAndConfirmOwner(t, app, other)
	assertStatus(t, app.request(t, "GET", path, nil), 404)
	assertStatus(t, serviceRawRequest(t, app, "GET", path, nil, "", false, false), 401)
	createProfessionalForList(t, app, "Ana", "get-prof@example.test")
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "get-prof@example.test", "password": "senha-temporaria"}), 200)
	assertStatus(t, app.request(t, "POST", "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), 204)
	assertStatus(t, app.request(t, "GET", path, nil), 403)
}

func TestServicePartialUpdateAndAtomicRefusals(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	record := createCatalogService(t, app, "Corte", 100)
	path := "/api/v1/services/" + record.ID
	if _, err := integrationPool.Exec(context.Background(), `UPDATE services SET active=false WHERE id=$1`, record.ID); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"name": " Barba ", "duration_minutes": 17}
	r := app.request(t, "PATCH", path, body)
	assertStatus(t, r, 200)
	var got serviceRecord
	decodeJSON(t, r, &got)
	if got.Name != "Barba" || got.PriceCents != 100 || got.DurationMinutes != 17 || got.Active || got.ID != record.ID {
		t.Fatalf("updated=%+v", got)
	}
	r = app.request(t, "PATCH", path, map[string]any{"duration_minutes": nil})
	assertStatus(t, r, 200)
	decodeJSON(t, r, &got)
	if got.DurationMinutes != 30 {
		t.Fatalf("reset=%+v", got)
	}
	for _, invalid := range []map[string]any{{}, {"name": nil}, {"price_cents": nil}, {"price_cents": 0.1}, {"name": "changed", "duration_minutes": 0}, {"active": true}, {"establishment_id": "foreign"}} {
		assertStatus(t, app.request(t, "PATCH", path, invalid), 400)
	}
	r = app.request(t, "GET", path, nil)
	assertStatus(t, r, 200)
	decodeJSON(t, r, &got)
	if got.Name != "Barba" || got.PriceCents != 100 || got.Active {
		t.Fatalf("invalid changed state=%+v", got)
	}
	assertStatus(t, serviceRawRequest(t, app, "PATCH", path, body, trustedOrigin, false, true), 403)
	assertStatus(t, serviceRawRequest(t, app, "PATCH", path, body, "https://external.example.test", true, true), 403)
	assertStatus(t, serviceRawRequest(t, app, "PATCH", path, body, trustedOrigin, true, false), 401)
	var wg sync.WaitGroup
	for _, change := range []map[string]any{{"name": "Novo nome"}, {"price_cents": 200}} {
		wg.Add(1)
		go func(payload map[string]any) {
			defer wg.Done()
			response := app.request(t, "PATCH", path, payload)
			if response.StatusCode != 200 {
				t.Errorf("concurrent patch=%d", response.StatusCode)
			}
		}(change)
	}
	wg.Wait()
	r = app.request(t, "GET", path, nil)
	assertStatus(t, r, 200)
	decodeJSON(t, r, &got)
	if got.Name != "Novo nome" || got.PriceCents != 200 {
		t.Fatalf("lost update=%+v", got)
	}
	assertStatus(t, app.request(t, "PATCH", "/api/v1/services/771d0b88-237b-4f35-89ad-850ffbd894d3", body), 404)
	other := defaultOwnerRegistration()
	other.Email = "patch-other@example.test"
	other.Slug = "patch-other"
	registerAndConfirmOwner(t, app, other)
	assertStatus(t, app.request(t, "PATCH", path, body), 404)
	own := createCatalogService(t, app, "Outro", 0)
	expireServiceTrial(t)
	assertStatus(t, app.request(t, "PATCH", "/api/v1/services/"+own.ID, body), 403)
}
