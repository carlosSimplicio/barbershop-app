//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestPasswordRecoveryTokensKeepOneActiveHashPerUser(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerOwner(t, application, registration)

	ctx := context.Background()
	var userID string
	if err := integrationPool.QueryRow(ctx,
		`SELECT id FROM users WHERE normalized_email = $1`, registration.Email,
	).Scan(&userID); err != nil {
		t.Fatalf("find registered owner: %v", err)
	}

	createdAt := time.Now().UTC()
	expiresAt := createdAt.Add(time.Hour)
	firstHash := bytes.Repeat([]byte{0x11}, 32)
	secondHash := bytes.Repeat([]byte{0x22}, 32)
	insert := func(hash []byte) error {
		_, err := integrationPool.Exec(ctx, `
			INSERT INTO password_recovery_tokens (user_id, token_hash, expires_at, created_at)
			VALUES ($1, $2, $3, $4)
		`, userID, hash, expiresAt, createdAt)
		return err
	}

	if err := insert(firstHash); err != nil {
		t.Fatalf("insert first recovery token: %v", err)
	}
	if err := insert(secondHash); err == nil {
		t.Fatal("accepted two active recovery tokens for one user")
	}
	if _, err := integrationPool.Exec(ctx,
		`UPDATE password_recovery_tokens SET invalidated_at = $2 WHERE user_id = $1`,
		userID, createdAt.Add(time.Minute),
	); err != nil {
		t.Fatalf("invalidate first recovery token: %v", err)
	}
	if err := insert(secondHash); err != nil {
		t.Fatalf("replacing inactive recovery token: %v", err)
	}
	if err := insert(firstHash); err == nil {
		t.Fatal("accepted a reused recovery token hash")
	}
	if _, err := integrationPool.Exec(ctx, `
		UPDATE password_recovery_tokens SET consumed_at = $2 WHERE user_id = $1 AND token_hash = $3
	`, userID, createdAt.Add(2*time.Minute), secondHash); err != nil {
		t.Fatalf("consume second recovery token: %v", err)
	}
	if err := insert([]byte("short")); err == nil {
		t.Fatal("accepted a recovery token hash shorter than SHA-256")
	}
	if _, err := integrationPool.Exec(ctx, `
		INSERT INTO password_recovery_tokens (user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $3)
	`, userID, bytes.Repeat([]byte{0x33}, 32), createdAt); err == nil {
		t.Fatal("accepted a recovery token expired at creation")
	}
}
