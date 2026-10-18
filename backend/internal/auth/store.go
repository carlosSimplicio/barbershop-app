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

func (store *Store) FindUserByID(ctx context.Context, userID string) (User, error) {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return User{}, ErrUserNotFound
	}
	row, err := store.queries.GetAuthUserByID(ctx, parsedUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get authentication user: %w", err)
	}
	return userFromIDRow(row), nil
}

func (store *Store) ListRateLimitEvents(ctx context.Context, scope, dimension string, keyHash []byte, since time.Time) ([]time.Time, error) {
	rows, err := store.queries.ListLoginAttemptsSince(ctx, db.ListLoginAttemptsSinceParams{
		Scope: scope, Dimension: dimension, KeyHash: keyHash, AttemptedAt: timestamptz(since),
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

func (store *Store) RecordRateLimitEvents(ctx context.Context, scope string, keys map[string][]byte, attemptedAt time.Time) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin authentication attempt transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	for dimension, key := range keys {
		if err := queries.RecordLoginAttempt(ctx, db.RecordLoginAttemptParams{
			Scope: scope, Dimension: dimension, KeyHash: key, AttemptedAt: timestamptz(attemptedAt),
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

func (store *Store) RotateRefreshToken(ctx context.Context, currentHash, replacementHash []byte, now, expiresAt time.Time) (User, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	current, err := queries.GetRefreshTokenForUpdate(ctx, currentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidSession
	}
	if err != nil {
		return User{}, fmt.Errorf("get refresh token: %w", err)
	}
	if current.ConsumedAt.Valid {
		if err := revokeSession(ctx, queries, current.SessionID, now); err != nil {
			return User{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, fmt.Errorf("commit reused refresh revocation: %w", err)
		}
		return User{}, ErrRefreshTokenReused
	}
	if current.TokenRevokedAt.Valid || current.SessionRevokedAt.Valid || !current.TokenExpiresAt.Time.After(now) || !current.IdleExpiresAt.Time.After(now) {
		return User{}, ErrInvalidSession
	}

	userRow, err := queries.GetAuthUserByID(ctx, current.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidSession
	}
	if err != nil {
		return User{}, fmt.Errorf("get refresh user: %w", err)
	}
	user := userFromIDRow(userRow)
	if !userCanAuthenticate(user) {
		if err := revokeSession(ctx, queries, current.SessionID, now); err != nil {
			return User{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, fmt.Errorf("commit invalid user session revocation: %w", err)
		}
		return User{}, ErrInvalidSession
	}
	replacement, err := queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		SessionID: current.SessionID, TokenHash: replacementHash, ExpiresAt: timestamptz(expiresAt),
	})
	if err != nil {
		return User{}, fmt.Errorf("create replacement refresh token: %w", err)
	}
	if err := queries.ConsumeRefreshToken(ctx, db.ConsumeRefreshTokenParams{
		ID: current.TokenID, ConsumedAt: timestamptz(now), ReplacedByID: replacement.ID,
	}); err != nil {
		return User{}, fmt.Errorf("consume refresh token: %w", err)
	}
	if err := queries.ExtendAuthSession(ctx, db.ExtendAuthSessionParams{
		ID: current.SessionID, IdleExpiresAt: timestamptz(expiresAt), UpdatedAt: timestamptz(now),
	}); err != nil {
		return User{}, fmt.Errorf("extend authentication session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit refresh transaction: %w", err)
	}
	return user, nil
}

func (store *Store) RevokeSessionByRefreshToken(ctx context.Context, refreshHash []byte, now time.Time) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin logout transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	current, err := queries.GetRefreshTokenForUpdate(ctx, refreshHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get logout refresh token: %w", err)
	}
	if err := revokeSession(ctx, queries, current.SessionID, now); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout transaction: %w", err)
	}
	return nil
}

func (store *Store) ChangeTemporaryPassword(ctx context.Context, userID string, expectedCredentialVersion int64, passwordHash string, refreshHash []byte, now, expiresAt time.Time) (int64, error) {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return 0, ErrPasswordChangeDenied
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin password change transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	user, err := queries.GetAuthUserByIDForUpdate(ctx, parsedUserID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!user.Active || !user.PasswordChangeRequired || user.CredentialVersion != expectedCredentialVersion)) {
		return 0, ErrPasswordChangeDenied
	}
	if err != nil {
		return 0, fmt.Errorf("lock password user: %w", err)
	}
	credentialVersion, err := queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID: parsedUserID, PasswordHash: passwordHash, UpdatedAt: timestamptz(now), CredentialVersion: expectedCredentialVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPasswordChangeDenied
	}
	if err != nil {
		return 0, fmt.Errorf("update temporary password: %w", err)
	}
	if err := queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{UserID: parsedUserID, UpdatedAt: timestamptz(now)}); err != nil {
		return 0, fmt.Errorf("revoke user sessions: %w", err)
	}
	if err := queries.RevokeAllUserRefreshTokens(ctx, db.RevokeAllUserRefreshTokensParams{UserID: parsedUserID, RevokedAt: timestamptz(now)}); err != nil {
		return 0, fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	session, err := queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{UserID: parsedUserID, IdleExpiresAt: timestamptz(expiresAt)})
	if err != nil {
		return 0, fmt.Errorf("create post-password session: %w", err)
	}
	if _, err := queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		SessionID: session.ID, TokenHash: refreshHash, ExpiresAt: timestamptz(expiresAt),
	}); err != nil {
		return 0, fmt.Errorf("create post-password refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit password change transaction: %w", err)
	}
	return credentialVersion, nil
}

func revokeSession(ctx context.Context, queries *db.Queries, sessionID pgtype.UUID, now time.Time) error {
	if err := queries.RevokeAuthSession(ctx, db.RevokeAuthSessionParams{ID: sessionID, UpdatedAt: timestamptz(now)}); err != nil {
		return fmt.Errorf("revoke authentication session: %w", err)
	}
	if err := queries.RevokeRefreshTokensBySession(ctx, db.RevokeRefreshTokensBySessionParams{SessionID: sessionID, RevokedAt: timestamptz(now)}); err != nil {
		return fmt.Errorf("revoke session refresh tokens: %w", err)
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

func userFromIDRow(row db.GetAuthUserByIDRow) User {
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
