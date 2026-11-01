//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

type professionalSecurityState struct {
	Active, Required       bool
	Version                int64
	Hash, Name, Email      string
	Sessions, LiveSessions int
}

func readProfessionalSecurityState(t *testing.T, id string) professionalSecurityState {
	t.Helper()
	var state professionalSecurityState
	err := integrationPool.QueryRow(context.Background(), `SELECT u.active,u.password_change_required,u.credential_version,c.password_hash,p.name,u.email,
 (SELECT count(*) FROM auth_sessions s WHERE s.user_id=u.id),
 (SELECT count(*) FROM auth_sessions s WHERE s.user_id=u.id AND s.revoked_at IS NULL)
 FROM users u JOIN user_credentials c ON c.user_id=u.id JOIN professionals p ON p.user_id=u.id WHERE u.id=$1`, id).Scan(&state.Active, &state.Required, &state.Version, &state.Hash, &state.Name, &state.Email, &state.Sessions, &state.LiveSessions)
	if err != nil {
		t.Fatal(err)
	}
	return state
}
func independentClient(t *testing.T, app *testApplication) *testApplication {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	peer := *app
	peer.client = &http.Client{Transport: app.client.Transport, Jar: jar}
	return &peer
}
func loginProfessional(t *testing.T, app *testApplication) {
	t.Helper()
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusOK)
}
func createManagementProfessional(t *testing.T, app *testApplication) string {
	t.Helper()
	registerAndConfirmOwner(t, app, defaultOwnerRegistration())
	createProfessionalForList(t, app, "Ana", "ana@example.test")
	return getProfessionalPage(t, app, "/api/v1/professionals").Items[0].ID
}
func TestProfessionalDeactivationRevokesAllSessionsAndIsRepeatable(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	first, second := independentClient(t, app), independentClient(t, app)
	loginProfessional(t, first)
	loginProfessional(t, second)
	before := readProfessionalSecurityState(t, id)
	path := "/api/v1/professionals/" + id + "/deactivation"
	response := app.request(t, http.MethodPost, path, nil)
	assertStatus(t, response, http.StatusOK)
	var record struct {
		ID, Name, Email string
		Active          bool
	}
	decodeJSON(t, response, &record)
	after := readProfessionalSecurityState(t, id)
	if record.ID != id || record.Active || record.Name != before.Name || record.Email != before.Email || after.Active || after.Version != before.Version+1 || after.Hash != before.Hash || after.Name != before.Name || after.Email != before.Email || after.Sessions != 2 || after.LiveSessions != 0 {
		t.Fatal("deactivation did not preserve identity and atomically revoke all access")
	}
	assertStatus(t, first.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusUnauthorized)
	assertStatus(t, second.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusUnauthorized)
	assertStatus(t, first.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusUnauthorized)
	assertStatus(t, app.request(t, http.MethodPost, path, nil), http.StatusOK)
	if readProfessionalSecurityState(t, id) != after {
		t.Fatal("repeated deactivation changed stored state")
	}
}
func failProfessionalSessionRevocation(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := integrationPool.Exec(ctx, `CREATE FUNCTION integration_fail_professional_revoke() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced revoke failure'; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := integrationPool.Exec(ctx, `CREATE TRIGGER integration_fail_professional_revoke BEFORE UPDATE ON auth_sessions FOR EACH ROW EXECUTE FUNCTION integration_fail_professional_revoke()`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = integrationPool.Exec(ctx, `DROP TRIGGER IF EXISTS integration_fail_professional_revoke ON auth_sessions`)
		_, _ = integrationPool.Exec(ctx, `DROP FUNCTION IF EXISTS integration_fail_professional_revoke()`)
	})
}
func TestProfessionalDeactivationRollsBackOnRevocationFailure(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	before := readProfessionalSecurityState(t, id)
	failProfessionalSessionRevocation(t)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/deactivation", nil), http.StatusInternalServerError)
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("partial deactivation survived rollback")
	}
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)
}
func TestProfessionalDeactivationRequiresTenantOwnerCSRFAndWriteAccess(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	path := "/api/v1/professionals/" + id + "/deactivation"
	before := readProfessionalSecurityState(t, id)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, nil, trustedOrigin, false, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, nil, "https://other.example.test", true, true), http.StatusForbidden)
	assertStatus(t, catalogRawRequest(t, app, http.MethodPost, path, nil, trustedOrigin, true, false), http.StatusUnauthorized)
	assertStatus(t, app.request(t, http.MethodPost, path, map[string]any{}), http.StatusBadRequest)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	assertStatus(t, peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertStatus(t, peer.request(t, http.MethodPost, path, nil), http.StatusForbidden)
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role='owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+ownerID+"/deactivation", nil), http.StatusNotFound)
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, app, other)
	assertStatus(t, app.request(t, http.MethodPost, path, nil), http.StatusNotFound)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), http.StatusOK)
	expireCatalogTrial(t)
	response := app.request(t, http.MethodPost, path, nil)
	assertStatus(t, response, http.StatusForbidden)
	var failure struct{ Code string }
	decodeJSON(t, response, &failure)
	if failure.Code != "read_only" {
		t.Fatal("expected read_only")
	}
	after := readProfessionalSecurityState(t, id)
	if !after.Active || after.Name != before.Name || after.Email != before.Email {
		t.Fatal("refused deactivation changed identity/state")
	}
}
func TestProfessionalDeactivationSerializesAgainstLoginAndPasswordChange(t *testing.T) {
	for _, operation := range []string{"login", "password change"} {
		t.Run(operation, func(t *testing.T) {
			app := newTestApplication(t)
			id := createManagementProfessional(t, app)
			peer := independentClient(t, app)
			if operation == "password change" {
				loginProfessional(t, peer)
			}
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
			deactivated := make(chan *http.Response, 1)
			go func() {
				deactivated <- app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/deactivation", nil)
			}()
			waitForBlockedDatabaseQuery(t, ctx, "LockProfessionalUser")
			raced := make(chan *http.Response, 1)
			go func() {
				if operation == "login" {
					raced <- peer.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"})
				} else {
					raced <- peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"})
				}
			}()
			waitForBlockedDatabaseQuery(t, ctx, "LockAuthSessionUser")
			if err = transaction.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case response := <-deactivated:
				assertStatus(t, response, http.StatusOK)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			select {
			case response := <-raced:
				expected := http.StatusUnauthorized
				if operation == "password change" {
					expected = http.StatusForbidden
				}
				assertStatus(t, response, expected)
				for _, cookie := range response.Cookies() {
					if cookie.MaxAge > 0 {
						t.Fatal("racing operation issued success cookies")
					}
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			after := readProfessionalSecurityState(t, id)
			if after.Active || after.LiveSessions != 0 || after.Version != 2 {
				t.Fatal("race left valid access after deactivation")
			}
		})
	}
}
