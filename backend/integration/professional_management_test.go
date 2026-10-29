//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestProfessionalGetReadsBothStatesAfterExpirationAndLimitsProjection(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	id := getProfessionalPage(t, application, "/api/v1/professionals").Items[0].ID
	path := "/api/v1/professionals/" + id
	for _, active := range []bool{true, false} {
		if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET active = $1 WHERE id = $2`, active, id); err != nil {
			t.Fatal(err)
		}
		response := application.request(t, http.MethodGet, path, nil)
		assertStatus(t, response, http.StatusOK)
		var record map[string]json.RawMessage
		decodeJSON(t, response, &record)
		if len(record) != 4 || string(record["id"]) != `"`+id+`"` || string(record["name"]) != `"Ana"` || string(record["email"]) != `"ana@example.test"` || string(record["active"]) != map[bool]string{true: "true", false: "false"}[active] {
			t.Fatal("unexpected professional projection")
		}
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at=$1, ends_at=$2`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, application.request(t, http.MethodGet, path, nil), http.StatusOK)
}

func TestProfessionalGetRequiresOwnerTenantAndProfessionalProfile(t *testing.T) {
	application := newTestApplication(t)
	path := "/api/v1/professionals/00000000-0000-4000-8000-000000000001"
	assertStatus(t, application.request(t, http.MethodGet, path, nil), http.StatusUnauthorized)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	assertStatus(t, application.request(t, http.MethodGet, path, nil), http.StatusNotFound)
	assertStatus(t, application.request(t, http.MethodGet, "/api/v1/professionals/invalid", nil), http.StatusBadRequest)
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role='owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, application.request(t, http.MethodGet, "/api/v1/professionals/"+ownerID, nil), http.StatusNotFound)
	createProfessionalForList(t, application, "Ana", "ana@example.test")
	id := getProfessionalPage(t, application, "/api/v1/professionals").Items[0].ID
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, application, other)
	assertStatus(t, application.request(t, http.MethodGet, "/api/v1/professionals/"+id, nil), http.StatusNotFound)
	assertStatus(t, application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusOK)
	assertStatus(t, application.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	response := application.request(t, http.MethodGet, "/api/v1/professionals/"+id, nil)
	assertStatus(t, response, http.StatusForbidden)
	var failure struct {
		Message string `json:"message"`
	}
	decodeJSON(t, response, &failure)
	if failure.Message != "Você não pode consultar este profissional." {
		t.Fatalf("unexpected consultation refusal: %q", failure.Message)
	}
}
