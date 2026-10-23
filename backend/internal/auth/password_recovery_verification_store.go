package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5"
)

func (store *Store) ResetOwnerPassword(ctx context.Context, tokenHash []byte, newPassword string, passwords PasswordManager) error {
	candidate, err := store.queries.GetPasswordRecoveryCandidate(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidPasswordRecovery
	}
	if err != nil {
		return fmt.Errorf("find password recovery candidate: %w", err)
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password recovery verification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	user, err := queries.GetAuthUserByIDForUpdate(ctx, candidate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidPasswordRecovery
	}
	if err != nil {
		return fmt.Errorf("lock password recovery owner: %w", err)
	}
	if user.Role != "owner" || !user.Active || !user.EmailConfirmedAt.Valid {
		return ErrInvalidPasswordRecovery
	}
	token, err := queries.GetPasswordRecoveryForUpdate(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidPasswordRecovery
	}
	if err != nil {
		return fmt.Errorf("lock password recovery token: %w", err)
	}
	if token.UserID != user.ID || !token.ExpiresAt.Time.After(time.Now().UTC()) || token.ConsumedAt.Valid || token.InvalidatedAt.Valid {
		return ErrInvalidPasswordRecovery
	}
	unchanged, err := passwords.Verify(newPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify previous owner password: %w", err)
	}
	if unchanged {
		return ErrPasswordUnchanged
	}
	newHash, err := passwords.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash recovered owner password: %w", err)
	}
	now := time.Now().UTC()
	consumed, err := queries.ConsumePasswordRecoveryToken(ctx, db.ConsumePasswordRecoveryTokenParams{
		ID: token.ID, UserID: user.ID, ConsumedAt: timestamptz(now),
	})
	if err != nil {
		return fmt.Errorf("consume password recovery token: %w", err)
	}
	if consumed != 1 {
		return ErrInvalidPasswordRecovery
	}
	rowsAdvanced, err := queries.AdvanceRecoveredOwnerCredentialVersion(ctx, db.AdvanceRecoveredOwnerCredentialVersionParams{
		ID: user.ID, UpdatedAt: timestamptz(now),
	})
	if err != nil {
		return fmt.Errorf("advance owner credential version: %w", err)
	}
	if rowsAdvanced != 1 {
		return ErrInvalidPasswordRecovery
	}
	rowsUpdated, err := queries.UpdateRecoveredOwnerPasswordHash(ctx, db.UpdateRecoveredOwnerPasswordHashParams{
		UserID: user.ID, PasswordHash: newHash, UpdatedAt: timestamptz(now),
	})
	if err != nil {
		return fmt.Errorf("update recovered owner password: %w", err)
	}
	if rowsUpdated != 1 {
		return fmt.Errorf("update recovered owner password: missing credential")
	}
	if err := queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{UserID: user.ID, UpdatedAt: timestamptz(now)}); err != nil {
		return fmt.Errorf("revoke recovered owner sessions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password recovery verification transaction: %w", err)
	}
	return nil
}
