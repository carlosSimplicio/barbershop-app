package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5"
)

func (store *Store) ResetOwnerPassword(ctx context.Context, tokenHash []byte, prepare RecoveredPasswordPreparer) error {
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
	token, err := queries.GetPasswordRecoveryForUpdate(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidPasswordRecovery
	}
	if err != nil {
		return fmt.Errorf("lock password recovery token: %w", err)
	}
	newHash, err := prepare(userFromIDRow(db.GetAuthUserByIDRow(user)), PasswordRecoveryState{
		UserID: uuidString(token.UserID), ExpiresAt: token.ExpiresAt.Time,
		Consumed: token.ConsumedAt.Valid, Invalidated: token.InvalidatedAt.Valid,
	})
	if err != nil {
		return err
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
