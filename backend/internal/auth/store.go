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

func (store *Store) FindActiveSessionUser(ctx context.Context, sessionID string, now time.Time) (User, error) {
	parsedSessionID, err := parseUUID(sessionID)
	if err != nil {
		return User{}, ErrInvalidSession
	}
	row, err := store.queries.GetActiveSessionUser(ctx, db.GetActiveSessionUserParams{ID: parsedSessionID, ExpiresAt: timestamptz(now)})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidSession
	}
	if err != nil {
		return User{}, fmt.Errorf("get active session user: %w", err)
	}
	return userFromActiveSessionRow(row), nil
}

func (store *Store) CreateSession(ctx context.Context, userID, sessionID string, expiresAt time.Time) error {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return err
	}
	parsedSessionID, err := parseUUID(sessionID)
	if err != nil {
		return err
	}
	_, err = store.queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{ID: parsedSessionID, UserID: parsedUserID, ExpiresAt: timestamptz(expiresAt)})
	if err != nil {
		return fmt.Errorf("create authentication session: %w", err)
	}
	return nil
}

func (store *Store) RevokeSession(ctx context.Context, sessionID, userID string, now time.Time) error {
	parsedSessionID, err := parseUUID(sessionID)
	if err != nil {
		return ErrInvalidSession
	}
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return ErrInvalidSession
	}
	if err := store.queries.RevokeAuthSession(ctx, db.RevokeAuthSessionParams{
		ID: parsedSessionID, UserID: parsedUserID, UpdatedAt: timestamptz(now),
	}); err != nil {
		return fmt.Errorf("revoke authentication session: %w", err)
	}
	return nil
}

func (store *Store) ChangeTemporaryPassword(ctx context.Context, userID string, expectedCredentialVersion int64, passwordHash, sessionID string, now, expiresAt time.Time) error {
	parsedUserID, err := parseUUID(userID)
	if err != nil {
		return ErrPasswordChangeDenied
	}
	parsedSessionID, err := parseUUID(sessionID)
	if err != nil {
		return ErrInvalidSession
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password change transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	user, err := queries.GetAuthUserByIDForUpdate(ctx, parsedUserID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!user.Active || !user.PasswordChangeRequired || user.CredentialVersion != expectedCredentialVersion)) {
		return ErrPasswordChangeDenied
	}
	if err != nil {
		return fmt.Errorf("lock password user: %w", err)
	}
	_, err = queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID: parsedUserID, PasswordHash: passwordHash, UpdatedAt: timestamptz(now), CredentialVersion: expectedCredentialVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPasswordChangeDenied
	}
	if err != nil {
		return fmt.Errorf("update temporary password: %w", err)
	}
	if err := queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{UserID: parsedUserID, UpdatedAt: timestamptz(now)}); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	_, err = queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{ID: parsedSessionID, UserID: parsedUserID, ExpiresAt: timestamptz(expiresAt)})
	if err != nil {
		return fmt.Errorf("create post-password session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password change transaction: %w", err)
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

func userFromActiveSessionRow(row db.GetActiveSessionUserRow) User {
	return User{
		ID: uuidString(row.ID), EstablishmentID: uuidString(row.EstablishmentID), Email: row.Email,
		NormalizedEmail: row.NormalizedEmail, Role: row.Role, Active: row.Active,
		EmailConfirmed: row.EmailConfirmedAt.Valid, CredentialVersion: row.CredentialVersion,
		PasswordChangeRequired: row.PasswordChangeRequired,
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
