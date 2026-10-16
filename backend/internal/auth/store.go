package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: db.New(pool)}
}

func (store *Store) FindUserByNormalizedEmail(ctx context.Context, normalizedEmail string) (User, error) {
	row, err := store.queries.GetAuthUserByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get authentication user: %w", err)
	}
	return userFromEmailRow(row), nil
}

func (store *Store) ListLoginAttempts(ctx context.Context, dimension string, keyHash []byte, since time.Time) ([]time.Time, error) {
	rows, err := store.queries.ListLoginAttemptsSince(ctx, db.ListLoginAttemptsSinceParams{
		Dimension: dimension, KeyHash: keyHash, AttemptedAt: timestamptz(since),
	})
	if err != nil {
		return nil, fmt.Errorf("list authentication attempts: %w", err)
	}
	attempts := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		attempts = append(attempts, row.Time)
	}
	return attempts, nil
}

func (store *Store) RecordLoginFailure(ctx context.Context, ipKey, emailKey []byte, attemptedAt time.Time) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin authentication attempt transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	for dimension, key := range map[string][]byte{"ip": ipKey, "email": emailKey} {
		if err := queries.RecordLoginAttempt(ctx, db.RecordLoginAttemptParams{
			Dimension: dimension, KeyHash: key, AttemptedAt: timestamptz(attemptedAt),
		}); err != nil {
			return fmt.Errorf("record authentication attempt: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit authentication attempt transaction: %w", err)
	}
	return nil
}

func (store *Store) CreateSession(ctx context.Context, userID string, refreshTokenHash []byte, expiresAt time.Time) error {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return err
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin authentication session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	session, err := queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{UserID: parsedUserID, IdleExpiresAt: timestamptz(expiresAt)})
	if err != nil {
		return fmt.Errorf("create authentication session: %w", err)
	}
	if _, err := queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		SessionID: session.ID, TokenHash: refreshTokenHash, ExpiresAt: timestamptz(expiresAt),
	}); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit authentication session transaction: %w", err)
	}
	return nil
}

func userFromEmailRow(row db.GetAuthUserByNormalizedEmailRow) User {
	return User{
		ID: uuidString(row.ID), EstablishmentID: uuidString(row.EstablishmentID), Email: row.Email,
		NormalizedEmail: row.NormalizedEmail, Role: row.Role, Active: row.Active,
		EmailConfirmed: row.EmailConfirmedAt.Valid, CredentialVersion: row.CredentialVersion,
		PasswordChangeRequired: row.PasswordChangeRequired, PasswordHash: row.PasswordHash,
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse UUID: %w", err)
	}
	return parsed, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	bytes := value.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
