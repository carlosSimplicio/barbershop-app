//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

type professionalOfferingRecord struct {
	offeringRecord
	Associated bool `json:"associated"`
}
type professionalServicesPage struct {
	Professional struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	} `json:"professional"`
	Items    []professionalOfferingRecord `json:"items"`
	Limit    int                          `json:"limit"`
	Offset   int                          `json:"offset"`
	HasMore  bool                         `json:"has_more"`
	CanWrite bool                         `json:"can_write"`
}

func getProfessionalServicesPage(t *testing.T, app *testApplication, path string) professionalServicesPage {
	t.Helper()
	response := app.request(t, http.MethodGet, path, nil)
	assertStatus(t, response, http.StatusOK)
	var page professionalServicesPage
	decodeJSON(t, response, &page)
	if page.Items == nil {
		t.Fatal("null items")
	}
	return page
}
func createServicesProfessional(t *testing.T, app *testApplication, name, email string) string {
	t.Helper()
	createProfessionalForList(t, app, name, email)
	return getProfessionalPage(t, app, "/api/v1/professionals").Items[0].ID
}

func TestProfessionalServicesListIncludesCatalogAndExplicitLinks(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	professionalID := createServicesProfessional(t, app, "Ana", "ana@example.test")
	path := "/api/v1/professionals/" + professionalID + "/services"
	empty := getProfessionalServicesPage(t, app, path)
	if len(empty.Items) != 0 || empty.HasMore || empty.Limit != 20 || empty.Offset != 0 || !empty.CanWrite || empty.Professional.ID != professionalID || empty.Professional.Name != "Ana" || !empty.Professional.Active {
		t.Fatalf("empty=%+v", empty)
	}
	first := createCatalogOffering(t, app, "Corte", 1250)
	second := createCatalogOffering(t, app, "Barba", 0)
	third := createCatalogOffering(t, app, "Acabamento", 100)
	// Equal timestamps exercise the UUID tie-breaker as well as bounded pagination.
	if _, err := integrationPool.Exec(context.Background(), `UPDATE services SET created_at='2026-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE services SET active=false WHERE id=$1`, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := integrationPool.Exec(context.Background(), `INSERT INTO professional_services (establishment_id,professional_id,service_id,active) SELECT establishment_id,$1,id,id=$2 FROM services WHERE id IN ($2,$3)`, professionalID, first.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	page := getProfessionalServicesPage(t, app, path+"?limit=2&offset=0")
	tail := getProfessionalServicesPage(t, app, path+"?limit=2&offset=2")
	if len(page.Items) != 2 || !page.HasMore || len(tail.Items) != 1 || tail.HasMore || tail.Offset != 2 {
		t.Fatalf("pagination=%+v %+v", page, tail)
	}
	all := append(page.Items, tail.Items...)
	for index, item := range all {
		if index > 0 && all[index-1].ID <= item.ID {
			t.Fatal("not ordered by descending ID")
		}
		switch item.ID {
		case first.ID:
			if !item.Associated || !item.Active || item.PriceCents != 1250 || item.DurationMinutes != 30 {
				t.Fatalf("first=%+v", item)
			}
		case second.ID:
			if item.Associated || item.Active {
				t.Fatalf("second=%+v", item)
			}
		case third.ID:
			if item.Associated {
				t.Fatalf("automatic link=%+v", item)
			}
		default:
			t.Fatalf("unexpected item=%+v", item)
		}
	}
	anotherID := createServicesProfessional(t, app, "Bia", "bia@example.test")
	for _, item := range getProfessionalServicesPage(t, app, "/api/v1/professionals/"+anotherID+"/services").Items {
		if item.Associated {
			t.Fatal("new professional automatically associated")
		}
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET active=false WHERE id=$1`, professionalID); err != nil {
		t.Fatal(err)
	}
	expireCatalogTrial(t)
	expired := getProfessionalServicesPage(t, app, path)
	if expired.CanWrite || expired.Professional.Active || len(expired.Items) != 3 {
		t.Fatalf("expired=%+v", expired)
	}
	if _, err := integrationPool.Exec(context.Background(), `DELETE FROM establishment_trials`); err != nil {
		t.Fatal(err)
	}
	if getProfessionalServicesPage(t, app, path).CanWrite {
		t.Fatal("missing trial allowed writing")
	}
}

func TestProfessionalServicesListRequiresOwnerAndTenant(t *testing.T) {
	app := newTestApplication(t)
	path := "/api/v1/professionals/" + uuid.NewString() + "/services"
	assertStatus(t, app.request(t, http.MethodGet, path, nil), http.StatusUnauthorized)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	assertStatus(t, app.request(t, http.MethodGet, path, nil), http.StatusNotFound)
	assertStatus(t, app.request(t, http.MethodGet, "/api/v1/professionals/invalid/services", nil), http.StatusBadRequest)
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role='owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, app.request(t, http.MethodGet, "/api/v1/professionals/"+ownerID+"/services", nil), http.StatusNotFound)
	id := createServicesProfessional(t, app, "Ana", "ana@example.test")
	path = "/api/v1/professionals/" + id + "/services"
	for _, query := range []string{"limit=0", "limit=101", "offset=-1", "offset=2147483648", "limit=invalid"} {
		assertStatus(t, app.request(t, http.MethodGet, path+"?"+query, nil), http.StatusBadRequest)
	}
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, app, other)
	createCatalogOffering(t, app, "Outro catálogo", 200)
	assertStatus(t, app.request(t, http.MethodGet, path, nil), http.StatusNotFound)
	otherID := createServicesProfessional(t, app, "Bia", "bia@example.test")
	otherPath := "/api/v1/professionals/" + otherID + "/services"
	if len(getProfessionalServicesPage(t, app, otherPath).Items) != 1 {
		t.Fatal("other tenant lost its catalog")
	}
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), http.StatusOK)
	if len(getProfessionalServicesPage(t, app, path).Items) != 0 {
		t.Fatal("foreign catalog exposed")
	}
	assertStatus(t, app.request(t, http.MethodGet, otherPath, nil), http.StatusNotFound)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusOK)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertStatus(t, app.request(t, http.MethodGet, path, nil), http.StatusForbidden)
}
