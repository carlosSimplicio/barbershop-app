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

func TestProfessionalNameEditPreservesIdentityCredentialsAndState(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	createProfessionalForList(t, app, "Ana", "ana@example.test")
	id := getProfessionalPage(t, app, "/api/v1/professionals").Items[0].ID
	path := "/api/v1/professionals/" + id
	var hashBefore string
	var versionBefore int64
	if err := integrationPool.QueryRow(context.Background(), `SELECT c.password_hash,u.credential_version FROM users u JOIN user_credentials c ON c.user_id=u.id WHERE u.id=$1`, id).Scan(&hashBefore, &versionBefore); err != nil {
		t.Fatal(err)
	}
	for _, active := range []bool{true, false} {
		if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET active=$1 WHERE id=$2`, active, id); err != nil {
			t.Fatal(err)
		}
		response := app.request(t, http.MethodPatch, path, map[string]any{"name": " Bia "})
		assertStatus(t, response, http.StatusOK)
		var record struct {
			ID, Name, Email string
			Active          bool
		}
		decodeJSON(t, response, &record)
		if record.ID != id || record.Name != "Bia" || record.Email != "ana@example.test" || record.Active != active {
			t.Fatal("edit changed unrelated professional data")
		}
	}
	var hashAfter string
	var versionAfter int64
	var required bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT c.password_hash,u.credential_version,u.password_change_required FROM users u JOIN user_credentials c ON c.user_id=u.id WHERE u.id=$1`, id).Scan(&hashAfter, &versionAfter, &required); err != nil {
		t.Fatal(err)
	}
	if hashAfter != hashBefore || versionAfter != versionBefore || !required {
		t.Fatal("edit changed credentials")
	}
	for _, body := range []any{map[string]any{}, map[string]any{"name": nil}, map[string]any{"name": 9}, map[string]any{"name": " "}, map[string]any{"name": "Ana\x00"}, map[string]any{"name": "Ana", "email": "changed@example.test"}, map[string]any{"name": "Ana", "active": true}} {
		assertStatus(t, app.request(t, http.MethodPatch, path, body), http.StatusBadRequest)
	}
	response := app.request(t, http.MethodPost, "/api/v1/professionals", map[string]any{"name": "Ana\x00", "email": "new@example.test", "temporary_password": "senha-temporaria"})
	assertStatus(t, response, http.StatusBadRequest)
	var failure struct{ Fields []struct{ Code string } }
	decodeJSON(t, response, &failure)
	if len(failure.Fields) != 1 || failure.Fields[0].Code != "invalid_name" {
		t.Fatal("NUL did not receive invalid_name")
	}
	assertStatus(t, catalogRawRequest(t, app, http.MethodPatch, path, map[string]string{"name": "Bia"}, trustedOrigin, false, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPatch, path, map[string]string{"name": "Bia"}, "https://other.example.test", true, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPatch, path, map[string]string{"name": "Bia"}, trustedOrigin, true, false), http.StatusUnauthorized)
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, app, other)
	assertStatus(t, app.request(t, http.MethodPatch, path, map[string]string{"name": "Other"}), http.StatusNotFound)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), http.StatusOK)
	expireCatalogTrial(t)
	response = app.request(t, http.MethodPatch, path, map[string]string{"name": "Expired"})
	assertStatus(t, response, http.StatusForbidden)
	var readOnly struct{ Code string }
	decodeJSON(t, response, &readOnly)
	if readOnly.Code != "read_only" {
		t.Fatal("expired write did not receive read_only")
	}
	response = app.request(t, http.MethodGet, path, nil)
	assertStatus(t, response, http.StatusOK)
	var record struct {
		Name   string
		Active bool
	}
	decodeJSON(t, response, &record)
	if record.Name != "Bia" || record.Active {
		t.Fatal("refused operation changed persisted name/state")
	}
}

func TestProfessionalNameEditPreservesConcurrentStateChange(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	createProfessionalForList(t, app, "Ana", "ana@example.test")
	id := getProfessionalPage(t, app, "/api/v1/professionals").Items[0].ID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()
	if _, err = transaction.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, id); err != nil {
		t.Fatal(err)
	}
	responses := make(chan *http.Response, 1)
	go func() {
		responses <- app.request(t, http.MethodPatch, "/api/v1/professionals/"+id, map[string]string{"name": "Bia"})
	}()
	waitForBlockedDatabaseQuery(t, ctx, "UpdateProfessionalName")
	if _, err = transaction.Exec(ctx, `UPDATE users SET active=false WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if err = transaction.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-responses:
		assertStatus(t, response, http.StatusOK)
		var result struct{ Active bool }
		decodeJSON(t, response, &result)
		if result.Active {
			t.Fatal("PATCH returned stale active state after concurrent deactivation")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	response := app.request(t, http.MethodGet, "/api/v1/professionals/"+id, nil)
	assertStatus(t, response, http.StatusOK)
	var record struct {
		Name   string
		Active bool
	}
	decodeJSON(t, response, &record)
	if record.Name != "Bia" || record.Active {
		t.Fatal("name edit lost concurrent state change")
	}
}

func TestProfessionalNameEditRejectsProfessionalRole(t *testing.T) {
	app := newTestApplication(t)
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	createProfessionalForList(t, app, "Ana", "ana@example.test")
	id := getProfessionalPage(t, app, "/api/v1/professionals").Items[0].ID
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusOK)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertStatus(t, app.request(t, http.MethodPatch, "/api/v1/professionals/"+id, map[string]string{"name": "Other"}), http.StatusForbidden)
}
