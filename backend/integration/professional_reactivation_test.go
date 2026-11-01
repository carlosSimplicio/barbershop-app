//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

func deactivateManagementProfessional(t *testing.T, app *testApplication, id string) {
	t.Helper()
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/deactivation", nil), http.StatusOK)
}
func TestProfessionalReactivationReusesPasswordWithNewHashAndRequiresChange(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	old := readProfessionalSecurityState(t, id)
	deactivateManagementProfessional(t, app, id)
	before := readProfessionalSecurityState(t, id)
	response := app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": "senha-temporaria"})
	assertStatus(t, response, http.StatusOK)
	var record map[string]any
	decodeJSON(t, response, &record)
	if len(record) != 4 || record["id"] != id || record["active"] != true {
		t.Fatal("reactivation exposed unexpected fields or failed to activate")
	}
	if len(response.Cookies()) != 0 {
		t.Fatal("reactivation authenticated the professional")
	}
	after := readProfessionalSecurityState(t, id)
	if !after.Active || !after.Required || after.Version != before.Version+1 || after.Hash == old.Hash || after.Name != old.Name || after.Email != old.Email || after.LiveSessions != 0 {
		t.Fatal("reactivation did not atomically replace credential and invalidate old access")
	}
	matches, err := app.passwords.Verify("senha-temporaria", after.Hash)
	if err != nil || !matches {
		t.Fatal("reused temporary password was not applied")
	}
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusUnauthorized)
	loginProfessional(t, peer)
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/establishment", nil), http.StatusForbidden)
	response = peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-temporaria"})
	assertStatus(t, response, http.StatusBadRequest)
	assertStatus(t, peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/establishment", nil), http.StatusOK)
}
func TestProfessionalReactivationRefusesActiveWithoutEffectsAndValidatesBody(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	path := "/api/v1/professionals/" + id + "/reactivation"
	before := readProfessionalSecurityState(t, id)
	response := app.request(t, http.MethodPost, path, map[string]string{"temporary_password": "outra-senha"})
	assertStatus(t, response, http.StatusConflict)
	var failure struct{ Code string }
	decodeJSON(t, response, &failure)
	if failure.Code != "professional_already_active" {
		t.Fatal("missing stable active conflict code")
	}
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("active conflict modified credentials or sessions")
	}
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)
	deactivateManagementProfessional(t, app, id)
	inactive := readProfessionalSecurityState(t, id)
	for _, body := range []any{map[string]any{}, map[string]any{"temporary_password": nil}, map[string]any{"temporary_password": 8}, map[string]any{"temporary_password": "short"}, map[string]any{"temporary_password": "senha temporaria"}, map[string]any{"temporary_password": "senha\u200btemporaria"}, map[string]any{"temporary_password": "senha-temporaria", "active": true}} {
		assertStatus(t, app.request(t, http.MethodPost, path, body), http.StatusBadRequest)
	}
	if readProfessionalSecurityState(t, id) != inactive {
		t.Fatal("invalid reactivation changed persisted state")
	}
}
func TestProfessionalReactivationRollsBackCredentialOnSessionFailure(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	deactivateManagementProfessional(t, app, id)
	before := readProfessionalSecurityState(t, id)
	failProfessionalSessionRevocation(t)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": "outra-senha"}), http.StatusInternalServerError)
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("partial reactivation survived rollback")
	}
}
func TestProfessionalReactivationAllowsOnlyOneConcurrentCredential(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	deactivateManagementProfessional(t, app, id)
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
	responses := make(chan *http.Response, 2)
	var workers sync.WaitGroup
	for _, password := range []string{"primeira-senha", "segunda-senha"} {
		workers.Add(1)
		go func() {
			defer workers.Done()
			responses <- app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": password})
		}()
	}
	waitForBlockedDatabaseQueryCount(t, ctx, "LockProfessionalUser", 2)
	if err = transaction.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	close(responses)
	counts := map[int]int{}
	for response := range responses {
		counts[response.StatusCode]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent reactivation statuses=%v", counts)
	}
	state := readProfessionalSecurityState(t, id)
	if state.Version != 3 || !state.Active || !state.Required {
		t.Fatal("concurrent reactivation replaced credential twice")
	}
	first, _ := app.passwords.Verify("primeira-senha", state.Hash)
	second, _ := app.passwords.Verify("segunda-senha", state.Hash)
	if first == second {
		t.Fatal("reactivation did not preserve exactly one credential")
	}
}
func TestProfessionalReactivationRequiresTenantOwnerCSRFAndWriteAccess(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	assertStatus(t, peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	path := "/api/v1/professionals/" + id + "/reactivation"
	body := map[string]string{"temporary_password": "senha-temporaria"}
	assertStatus(t, peer.request(t, http.MethodPost, path, body), http.StatusForbidden)
	deactivateManagementProfessional(t, app, id)
	before := readProfessionalSecurityState(t, id)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, body, trustedOrigin, false, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, body, "https://other.example.test", true, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, body, trustedOrigin, true, false), http.StatusUnauthorized)
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role='owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+ownerID+"/reactivation", body), http.StatusNotFound)
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, app, other)
	assertStatus(t, app.request(t, http.MethodPost, path, body), http.StatusNotFound)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), http.StatusOK)
	expireCatalogTrial(t)
	response := app.request(t, http.MethodPost, path, body)
	assertStatus(t, response, http.StatusForbidden)
	var failure struct{ Code string }
	decodeJSON(t, response, &failure)
	if failure.Code != "read_only" || readProfessionalSecurityState(t, id) != before {
		t.Fatal("refused reactivation changed data or lost read_only code")
	}
}
