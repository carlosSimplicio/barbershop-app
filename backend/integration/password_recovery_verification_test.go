//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"barbershop-app/backend/internal/auth/security"
)

const recoveryVerifyPath = "/api/v1/auth/password/recovery/verify"

func requestRecoveryToken(t *testing.T, application *testApplication, email string) string {
	t.Helper()
	response := application.request(t, http.MethodPost, "/api/v1/auth/password/recovery", map[string]string{"email": email})
	assertStatus(t, response, http.StatusAccepted)
	return application.messages.latestToken(t)
}

func TestPasswordRecoveryVerificationChangesPasswordAndRevokesOldSession(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	response := application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": registration.Email, "password": registration.Password})
	assertStatus(t, response, http.StatusOK)
	token := requestRecoveryToken(t, application, registration.Email)

	response = application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": token, "new_password": "nova-senha-segura"})
	assertStatus(t, response, http.StatusNoContent)
	if len(response.Header.Values("Set-Cookie")) != 0 {
		t.Fatal("recovery issued a session cookie")
	}
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)
	assertRecoveryInvalid(t, application, token)
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": registration.Email, "password": registration.Password})
	assertStatus(t, response, http.StatusUnauthorized)
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": registration.Email, "password": "nova-senha-segura"})
	assertStatus(t, response, http.StatusOK)

	var version int64
	var consumed, revoked bool
	var oldSessions int
	if err := integrationPool.QueryRow(context.Background(), `SELECT u.credential_version,
		(SELECT consumed_at IS NOT NULL FROM password_recovery_tokens WHERE user_id = u.id),
		(SELECT bool_and(revoked_at IS NOT NULL) FROM auth_sessions WHERE user_id = u.id AND created_at <
			(SELECT consumed_at FROM password_recovery_tokens WHERE user_id = u.id)),
		(SELECT count(*) FROM auth_sessions WHERE user_id = u.id AND created_at <
			(SELECT consumed_at FROM password_recovery_tokens WHERE user_id = u.id))
		FROM users u WHERE normalized_email = $1`, registration.Email).Scan(&version, &consumed, &revoked, &oldSessions); err != nil {
		t.Fatalf("load recovered owner state: %v", err)
	}
	if version != 2 || !consumed || !revoked || oldSessions != 2 {
		t.Fatalf("version=%d consumed=%t old sessions revoked=%t count=%d", version, consumed, revoked, oldSessions)
	}
}

func TestPasswordRecoveryVerificationRejectsReplacedExpiredAndUnchanged(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	oldToken := requestRecoveryToken(t, application, registration.Email)
	newToken := requestRecoveryToken(t, application, registration.Email)

	assertRecoveryInvalid(t, application, oldToken)
	response := application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": newToken, "new_password": "curta"})
	assertStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(readResponseBody(t, response), `"invalid_password"`) {
		t.Fatal("invalid new password did not produce a field error")
	}
	response = application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": newToken, "new_password": registration.Password})
	assertStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(readResponseBody(t, response), `"password_unchanged"`) {
		t.Fatal("reusing current password did not produce a field error")
	}
	if _, err := integrationPool.Exec(context.Background(), `UPDATE password_recovery_tokens SET created_at = now() - interval '2 hours', expires_at = now() - interval '1 hour' WHERE token_hash = $1`,
		security.HashOpaqueToken(newToken)); err != nil {
		t.Fatalf("expire recovery token: %v", err)
	}
	assertRecoveryInvalid(t, application, newToken)
	assertRecoveryInvalid(t, application, "malformed-token")
	nonexistentToken, err := security.NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	assertRecoveryInvalid(t, application, nonexistentToken)
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
}

func TestPasswordRecoveryVerificationRejectsOwnerWhoLostEligibility(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	token := requestRecoveryToken(t, application, registration.Email)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET active = false WHERE normalized_email = $1`, registration.Email); err != nil {
		t.Fatalf("deactivate recovery owner: %v", err)
	}
	assertRecoveryInvalid(t, application, token)
	var consumed bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT consumed_at IS NOT NULL FROM password_recovery_tokens WHERE token_hash = $1`, security.HashOpaqueToken(token)).Scan(&consumed); err != nil {
		t.Fatalf("load token after lost eligibility: %v", err)
	}
	if consumed {
		t.Fatal("ineligible owner consumed recovery token")
	}
}

func TestPasswordRecoveryVerificationDoesNotChangeAnotherOwner(t *testing.T) {
	application := newTestApplication(t)
	first := defaultOwnerRegistration()
	second := ownerRegistration{
		EstablishmentName: "Outra Barbearia", Slug: "outra-barbearia", TimeZone: "America/Sao_Paulo",
		Email: "other@example.test", Password: "senha-da-outra",
	}
	registerAndConfirmOwner(t, application, first)
	registerAndConfirmOwner(t, application, second)
	token := requestRecoveryToken(t, application, first.Email)
	response := application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": token, "new_password": "nova-senha-segura"})
	assertStatus(t, response, http.StatusNoContent)
	response = application.request(t, http.MethodGet, "/api/v1/auth/me", nil)
	assertStatus(t, response, http.StatusOK)
	var version int64
	var revoked bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT u.credential_version, s.revoked_at IS NOT NULL
		FROM users u JOIN auth_sessions s ON s.user_id = u.id WHERE u.normalized_email = $1`, second.Email).Scan(&version, &revoked); err != nil {
		t.Fatalf("load other owner state: %v", err)
	}
	if version != 1 || revoked {
		t.Fatalf("other owner's credential version = %d, session revoked = %t", version, revoked)
	}
}

func TestPasswordRecoveryVerificationRollsBackAfterSessionWriteFailure(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	token := requestRecoveryToken(t, application, registration.Email)
	ctx := context.Background()
	_, err := integrationPool.Exec(ctx, `CREATE FUNCTION integration_fail_recovery_revoke() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced revoke failure'; END $$`)
	if err != nil {
		t.Fatalf("create failure function: %v", err)
	}
	_, err = integrationPool.Exec(ctx, `CREATE TRIGGER integration_fail_recovery_revoke BEFORE UPDATE ON auth_sessions FOR EACH ROW EXECUTE FUNCTION integration_fail_recovery_revoke()`)
	if err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = integrationPool.Exec(ctx, `DROP TRIGGER IF EXISTS integration_fail_recovery_revoke ON auth_sessions`)
		_, _ = integrationPool.Exec(ctx, `DROP FUNCTION IF EXISTS integration_fail_recovery_revoke()`)
	})
	response := application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": token, "new_password": "nova-senha-segura"})
	assertStatus(t, response, http.StatusInternalServerError)
	var version int64
	var consumed bool
	var oldPasswordHash string
	if err := integrationPool.QueryRow(ctx, `SELECT u.credential_version, r.consumed_at IS NOT NULL, c.password_hash
		FROM users u JOIN user_credentials c ON c.user_id = u.id JOIN password_recovery_tokens r ON r.user_id = u.id
		WHERE u.normalized_email = $1`, registration.Email).Scan(&version, &consumed, &oldPasswordHash); err != nil {
		t.Fatalf("load state after forced failure: %v", err)
	}
	match, err := application.passwords.Verify(registration.Password, oldPasswordHash)
	if err != nil || version != 1 || consumed || !match {
		t.Fatalf("partial recovery survived rollback: version=%d consumed=%t match=%t error=%v", version, consumed, match, err)
	}
}

func TestPasswordRecoveryVerificationAllowsOneConcurrentUse(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	token := requestRecoveryToken(t, application, registration.Email)
	payload, err := json.Marshal(map[string]string{"token": token, "new_password": "nova-senha-segura"})
	if err != nil {
		t.Fatal(err)
	}
	statuses := make(chan int, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			request, requestErr := http.NewRequestWithContext(context.Background(), http.MethodPost, application.server.URL+recoveryVerifyPath, strings.NewReader(string(payload)))
			if requestErr != nil {
				statuses <- 0
				return
			}
			request.Header.Set("Origin", trustedOrigin)
			request.Header.Set("Content-Type", "application/json")
			response, responseErr := application.client.Do(request)
			if responseErr != nil {
				statuses <- 0
				return
			}
			_ = response.Body.Close()
			statuses <- response.StatusCode
		}()
	}
	group.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusNoContent] != 1 || counts[http.StatusBadRequest] != 1 {
		t.Fatalf("concurrent recovery statuses = %#v", counts)
	}
}

func assertRecoveryInvalid(t *testing.T, application *testApplication, token string) {
	t.Helper()
	response := application.request(t, http.MethodPost, recoveryVerifyPath, map[string]string{"token": token, "new_password": "nova-senha-segura"})
	assertStatus(t, response, http.StatusBadRequest)
	var failure map[string]any
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
		t.Fatalf("decode invalid token response: %v", err)
	}
	if requestID, ok := failure["request_id"].(string); !ok || requestID == "" {
		t.Fatalf("missing request ID: %+v", failure)
	}
	delete(failure, "request_id")
	want := map[string]any{"code": "invalid_password_recovery", "message": "Link inválido ou expirado."}
	if !reflect.DeepEqual(failure, want) {
		t.Fatalf("invalid token response differs: %+v", failure)
	}
}

func readResponseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
