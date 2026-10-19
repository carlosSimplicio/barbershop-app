package onboarding

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool              *pgxpool.Pool
	queries           *db.Queries
	now               func() time.Time
	beginRegistration func(context.Context) (registrationTransaction, error)
}

func NewStore(pool *pgxpool.Pool) *Store {
	store := &Store{pool: pool, queries: db.New(pool), now: time.Now}
	store.beginRegistration = func(ctx context.Context) (registrationTransaction, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &postgresRegistrationTransaction{tx: tx, queries: db.New(tx)}, nil
	}
	return store
}

type registrationTransaction interface {
	CreateEstablishment(context.Context, Registration) (pgtype.UUID, error)
	CreateOwner(context.Context, pgtype.UUID, Registration) (pgtype.UUID, error)
	CreateCredential(context.Context, pgtype.UUID, Registration) error
	CreateConfirmation(context.Context, pgtype.UUID, Registration) error
	Commit(context.Context) error
	Rollback(context.Context) error
}

type postgresRegistrationTransaction struct {
	tx      pgx.Tx
	queries *db.Queries
}

func (transaction *postgresRegistrationTransaction) CreateEstablishment(ctx context.Context, registration Registration) (pgtype.UUID, error) {
	return transaction.queries.CreateRegistrationEstablishment(ctx, db.CreateRegistrationEstablishmentParams{
		Name: registration.EstablishmentName, Slug: registration.Slug, TimeZone: registration.TimeZone,
		CreatedAt: timestamp(registration.CreatedAt),
	})
}

func (transaction *postgresRegistrationTransaction) CreateOwner(ctx context.Context, establishmentID pgtype.UUID, registration Registration) (pgtype.UUID, error) {
	return transaction.queries.CreateRegistrationOwner(ctx, db.CreateRegistrationOwnerParams{
		EstablishmentID: establishmentID, Email: registration.Email, NormalizedEmail: registration.NormalizedEmail,
		Role: registration.Role, CreatedAt: timestamp(registration.CreatedAt),
	})
}

func (transaction *postgresRegistrationTransaction) CreateCredential(ctx context.Context, userID pgtype.UUID, registration Registration) error {
	return transaction.queries.CreateRegistrationCredential(ctx, db.CreateRegistrationCredentialParams{
		UserID: userID, PasswordHash: registration.PasswordHash, CreatedAt: timestamp(registration.CreatedAt),
	})
}

func (transaction *postgresRegistrationTransaction) CreateConfirmation(ctx context.Context, userID pgtype.UUID, registration Registration) error {
	return transaction.queries.CreateRegistrationEmailConfirmation(ctx, db.CreateRegistrationEmailConfirmationParams{
		UserID: userID, TokenHash: registration.ConfirmationTokenHash,
		ExpiresAt: timestamp(registration.ConfirmationExpiresAt), CreatedAt: timestamp(registration.CreatedAt),
	})
}

func (transaction *postgresRegistrationTransaction) Commit(ctx context.Context) error {
	return transaction.tx.Commit(ctx)
}

func (transaction *postgresRegistrationTransaction) Rollback(ctx context.Context) error {
	return transaction.tx.Rollback(ctx)
}

func (store *Store) CreateRegistration(ctx context.Context, registration Registration) error {
	transaction, err := store.beginRegistration(ctx)
	if err != nil {
		return fmt.Errorf("begin registration transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	establishmentID, err := transaction.CreateEstablishment(ctx, registration)
	if err != nil {
		return registrationWriteError("create registration establishment", err)
	}
	userID, err := transaction.CreateOwner(ctx, establishmentID, registration)
	if err != nil {
		return registrationWriteError("create registration owner", err)
	}
	if err := transaction.CreateCredential(ctx, userID, registration); err != nil {
		return registrationWriteError("create registration credential", err)
	}
	if err := transaction.CreateConfirmation(ctx, userID, registration); err != nil {
		return registrationWriteError("create registration confirmation", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit registration transaction: %w", err)
	}
	return nil
}

func registrationWriteError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.ConstraintName {
		case "users_normalized_email_unique":
			return ErrEmailUnavailable
		case "establishments_slug_unique":
			return ErrSlugUnavailable
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
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
