//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestLoginRejectsIdentityChangedBeforeSessionPersistence(t *testing.T) {
	for _, change := range []string{
		`active=false`, `credential_version=credential_version+1`, `email_confirmed_at=NULL`,
		`role='professional'`, `password_change_required=true`,
	} {
		t.Run(change, func(t *testing.T) {
			app := newTestApplication(t)
			registerAndConfirmOwner(t, app, defaultOwnerRegistration())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			transaction, err := integrationPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = transaction.Rollback(context.Background()) }()
			var userID string
			if err = transaction.QueryRow(ctx, `SELECT id FROM users WHERE role='owner' FOR UPDATE`).Scan(&userID); err != nil {
				t.Fatal(err)
			}
			var before int
			if err = transaction.QueryRow(ctx, `SELECT count(*) FROM auth_sessions WHERE user_id=$1`, userID).Scan(&before); err != nil {
				t.Fatal(err)
			}
			responses := make(chan *http.Response, 1)
			go func() {
				responses <- app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"})
			}()
			waitForBlockedDatabaseQuery(t, ctx, "LockAuthSessionUser")
			if _, err = transaction.Exec(ctx, `UPDATE users SET `+change+` WHERE id=$1`, userID); err != nil {
				t.Fatal(err)
			}
			if err = transaction.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case response := <-responses:
				assertStatus(t, response, http.StatusUnauthorized)
				var failure struct{ Code string }
				decodeJSON(t, response, &failure)
				if failure.Code != "invalid_credentials" {
					t.Fatal("stale login did not use generic invalid credentials")
				}
				for _, cookie := range response.Cookies() {
					if cookie.MaxAge > 0 {
						t.Fatal("stale login issued success cookie")
					}
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var after int
			if err = integrationPool.QueryRow(ctx, `SELECT count(*) FROM auth_sessions WHERE user_id=$1`, userID).Scan(&after); err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatal("stale login persisted a session")
			}
		})
	}
}
