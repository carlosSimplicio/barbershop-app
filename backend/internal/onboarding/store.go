package onboarding

import (
	"context"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	now     func() time.Time
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: db.New(pool), now: time.Now}
}

func (store *Store) ReplaceEmailConfirmation(ctx context.Context, userID string) (EmailConfirmation, error) {
	var parsedUserID pgtype.UUID
	if err := parsedUserID.Scan(userID); err != nil {
		return EmailConfirmation{}, fmt.Errorf("parse confirmation user ID: %w", err)
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return EmailConfirmation{}, fmt.Errorf("begin email confirmation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	if _, err := queries.LockUserForEmailConfirmation(ctx, parsedUserID); err != nil {
		return EmailConfirmation{}, fmt.Errorf("lock email confirmation user: %w", err)
	}
	createdAt := store.now().UTC()
	confirmation, err := NewEmailConfirmation(createdAt)
	if err != nil {
		return EmailConfirmation{}, fmt.Errorf("generate email confirmation: %w", err)
	}
	if err := queries.InvalidateActiveEmailConfirmations(ctx, db.InvalidateActiveEmailConfirmationsParams{
		UserID: parsedUserID, InvalidatedAt: timestamp(createdAt),
	}); err != nil {
		return EmailConfirmation{}, fmt.Errorf("invalidate email confirmations: %w", err)
	}
	if _, err := queries.CreateEmailConfirmation(ctx, db.CreateEmailConfirmationParams{
		UserID: parsedUserID, TokenHash: confirmation.TokenHash,
		ExpiresAt: timestamp(confirmation.ExpiresAt), CreatedAt: timestamp(createdAt),
	}); err != nil {
		return EmailConfirmation{}, fmt.Errorf("create email confirmation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return EmailConfirmation{}, fmt.Errorf("commit email confirmation transaction: %w", err)
	}
	return confirmation, nil
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
