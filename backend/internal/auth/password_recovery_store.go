package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5"
)

func (store *Store) FindEligibleOwnerForPasswordRecovery(ctx context.Context, normalizedEmail string) (PasswordRecoveryAccount, bool, error) {
	row, err := store.queries.FindEligibleOwnerForPasswordRecovery(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordRecoveryAccount{}, false, nil
	}
	if err != nil {
		return PasswordRecoveryAccount{}, false, fmt.Errorf("find password recovery account: %w", err)
	}
	return PasswordRecoveryAccount{UserID: uuidString(row.ID), Email: row.Email}, true, nil
}

func (store *Store) ReplacePasswordRecoveryToken(ctx context.Context, userID string) (PasswordRecoveryToken, bool, error) {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("parse password recovery user ID: %w", err)
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("begin password recovery transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	if _, err := queries.LockEligibleOwnerForPasswordRecovery(ctx, parsedUserID); errors.Is(err, pgx.ErrNoRows) {
		return PasswordRecoveryToken{}, false, nil
	} else if err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("lock password recovery owner: %w", err)
	}
	createdAt := time.Now().UTC()
	token, err := newPasswordRecoveryToken(createdAt)
	if err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("generate password recovery token: %w", err)
	}
	if err := queries.InvalidateActivePasswordRecoveryTokens(ctx, db.InvalidateActivePasswordRecoveryTokensParams{UserID: parsedUserID, InvalidatedAt: timestamptz(createdAt)}); err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("invalidate password recovery tokens: %w", err)
	}
	if _, err := queries.CreatePasswordRecoveryToken(ctx, db.CreatePasswordRecoveryTokenParams{
		UserID: parsedUserID, TokenHash: token.TokenHash, ExpiresAt: timestamptz(token.ExpiresAt), CreatedAt: timestamptz(createdAt),
	}); err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("create password recovery token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PasswordRecoveryToken{}, false, fmt.Errorf("commit password recovery transaction: %w", err)
	}
	return token, true, nil
}
