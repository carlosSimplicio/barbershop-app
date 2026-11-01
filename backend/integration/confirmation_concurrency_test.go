//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestConfirmationVerificationWaitsForOwnerBeforeLockingToken(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	token := registerOwner(t, application, registration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID string
	if err := tx.QueryRow(ctx, "SELECT id::text FROM users WHERE normalized_email=$1 FOR UPDATE", registration.Email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	responses := make(chan *http.Response, 1)
	go func() {
		responses <- application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": token})
	}()
	waitForBlockedDatabaseQuery(t, ctx, "GetEmailConfirmation")
	// A resend already holding the owner must be able to invalidate its token.
	if _, err := tx.Exec(ctx, "UPDATE email_confirmations SET invalidated_at=now() WHERE user_id=$1 AND invalidated_at IS NULL AND consumed_at IS NULL", userID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-responses:
		assertStatus(t, response, http.StatusBadRequest)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	response := application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation", map[string]string{"email": registration.Email})
	assertStatus(t, response, http.StatusAccepted)
	response = application.request(t, http.MethodPost, "/api/v1/auth/email-confirmation/verify", map[string]string{"token": application.messages.latestToken(t)})
	assertStatus(t, response, http.StatusOK)
}

func waitForBlockedDatabaseQuery(t *testing.T, ctx context.Context, name string) {
	t.Helper()
	waitForBlockedDatabaseQueryCount(t, ctx, name, 1)
}

func waitForBlockedDatabaseQueryCount(t *testing.T, ctx context.Context, name string, expected int) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked int
		if err := integrationPool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE query LIKE '%' || $1 || '%' AND wait_event_type='Lock'", name).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked >= expected {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("database query did not reach the expected lock")
		}
	}
}
