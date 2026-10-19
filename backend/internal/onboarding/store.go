package onboarding

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"
	"barbershop-app/backend/internal/auth"

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
	beginConfirmation func(context.Context) (confirmationTransaction, error)
	beginVerification func(context.Context) (verificationTransaction, error)
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
	store.beginConfirmation = func(ctx context.Context) (confirmationTransaction, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &postgresConfirmationTransaction{tx: tx, queries: db.New(tx)}, nil
	}
	store.beginVerification = func(ctx context.Context) (verificationTransaction, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &postgresVerificationTransaction{tx: tx, queries: db.New(tx)}, nil
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

type confirmationTransaction interface {
	LockEligibleOwner(context.Context, pgtype.UUID) (bool, error)
	InvalidateActive(context.Context, pgtype.UUID, time.Time) error
	Create(context.Context, pgtype.UUID, EmailConfirmation, time.Time) error
	Commit(context.Context) error
	Rollback(context.Context) error
}

type postgresConfirmationTransaction struct {
	tx      pgx.Tx
	queries *db.Queries
}

type verificationRecord struct {
	ConfirmationID         pgtype.UUID
	UserID                 pgtype.UUID
	EstablishmentID        pgtype.UUID
	Email                  string
	Role                   string
	Active                 bool
	EmailConfirmed         bool
	CredentialVersion      int64
	PasswordChangeRequired bool
	ExpiresAt              time.Time
	Consumed               bool
	Invalidated            bool
}

type verificationTransaction interface {
	Lock(context.Context, []byte) (verificationRecord, bool, error)
	ConfirmUser(context.Context, pgtype.UUID, time.Time) (bool, error)
	Consume(context.Context, pgtype.UUID, time.Time) (bool, error)
	CreateTrial(context.Context, verificationRecord, TrialWindow) error
	CreateSession(context.Context, pgtype.UUID, []byte, time.Time) error
	Commit(context.Context) error
	Rollback(context.Context) error
}

type postgresVerificationTransaction struct {
	tx      pgx.Tx
	queries *db.Queries
}

func (transaction *postgresVerificationTransaction) Lock(ctx context.Context, tokenHash []byte) (verificationRecord, bool, error) {
	row, err := transaction.queries.GetEmailConfirmationForUpdate(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return verificationRecord{}, false, nil
	}
	if err != nil {
		return verificationRecord{}, false, err
	}
	return verificationRecord{
		ConfirmationID: row.ID, UserID: row.UserID, EstablishmentID: row.EstablishmentID, Email: row.Email,
		Role: row.Role, Active: row.Active, EmailConfirmed: row.EmailConfirmedAt.Valid,
		CredentialVersion: row.CredentialVersion, PasswordChangeRequired: row.PasswordChangeRequired,
		ExpiresAt: row.ExpiresAt.Time, Consumed: row.ConsumedAt.Valid, Invalidated: row.InvalidatedAt.Valid,
	}, true, nil
}

func (transaction *postgresVerificationTransaction) ConfirmUser(ctx context.Context, userID pgtype.UUID, at time.Time) (bool, error) {
	rows, err := transaction.queries.ConfirmUserEmail(ctx, db.ConfirmUserEmailParams{ID: userID, EmailConfirmedAt: timestamp(at)})
	return rows == 1, err
}

func (transaction *postgresVerificationTransaction) Consume(ctx context.Context, confirmationID pgtype.UUID, at time.Time) (bool, error) {
	rows, err := transaction.queries.ConsumeEmailConfirmation(ctx, db.ConsumeEmailConfirmationParams{ID: confirmationID, ConsumedAt: timestamp(at)})
	return rows == 1, err
}

func (transaction *postgresVerificationTransaction) CreateTrial(ctx context.Context, record verificationRecord, window TrialWindow) error {
	_, err := transaction.queries.CreateEstablishmentTrial(ctx, db.CreateEstablishmentTrialParams{
		StartsAt: timestamp(window.StartsAt), EndsAt: timestamp(window.EndsAt), OwnerUserID: record.UserID, EstablishmentID: record.EstablishmentID,
	})
	return err
}

func (transaction *postgresVerificationTransaction) CreateSession(ctx context.Context, userID pgtype.UUID, tokenHash []byte, expiresAt time.Time) error {
	session, err := transaction.queries.CreateAuthSession(ctx, db.CreateAuthSessionParams{UserID: userID, IdleExpiresAt: timestamp(expiresAt)})
	if err != nil {
		return err
	}
	_, err = transaction.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{SessionID: session.ID, TokenHash: tokenHash, ExpiresAt: timestamp(expiresAt)})
	return err
}

func (transaction *postgresVerificationTransaction) Commit(ctx context.Context) error {
	return transaction.tx.Commit(ctx)
}
func (transaction *postgresVerificationTransaction) Rollback(ctx context.Context) error {
	return transaction.tx.Rollback(ctx)
}

func (transaction *postgresConfirmationTransaction) LockEligibleOwner(ctx context.Context, userID pgtype.UUID) (bool, error) {
	_, err := transaction.queries.LockUserForEmailConfirmation(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (transaction *postgresConfirmationTransaction) InvalidateActive(ctx context.Context, userID pgtype.UUID, at time.Time) error {
	return transaction.queries.InvalidateActiveEmailConfirmations(ctx, db.InvalidateActiveEmailConfirmationsParams{
		UserID: userID, InvalidatedAt: timestamp(at),
	})
}

func (transaction *postgresConfirmationTransaction) Create(ctx context.Context, userID pgtype.UUID, confirmation EmailConfirmation, createdAt time.Time) error {
	_, err := transaction.queries.CreateEmailConfirmation(ctx, db.CreateEmailConfirmationParams{
		UserID: userID, TokenHash: confirmation.TokenHash,
		ExpiresAt: timestamp(confirmation.ExpiresAt), CreatedAt: timestamp(createdAt),
	})
	return err
}

func (transaction *postgresConfirmationTransaction) Commit(ctx context.Context) error {
	return transaction.tx.Commit(ctx)
}

func (transaction *postgresConfirmationTransaction) Rollback(ctx context.Context) error {
	return transaction.tx.Rollback(ctx)
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

func (store *Store) FindUnconfirmedOwnerByNormalizedEmail(ctx context.Context, normalizedEmail string) (ConfirmationAccount, bool, error) {
	row, err := store.queries.FindUnconfirmedOwnerByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmationAccount{}, false, nil
	}
	if err != nil {
		return ConfirmationAccount{}, false, fmt.Errorf("find email confirmation account: %w", err)
	}
	return ConfirmationAccount{UserID: uuidString(row.ID), Email: row.Email}, true, nil
}

func (store *Store) ReplaceEmailConfirmation(ctx context.Context, userID string) (EmailConfirmation, bool, error) {
	var parsedUserID pgtype.UUID
	if err := parsedUserID.Scan(userID); err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("parse confirmation user ID: %w", err)
	}
	transaction, err := store.beginConfirmation(ctx)
	if err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("begin email confirmation transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	eligible, err := transaction.LockEligibleOwner(ctx, parsedUserID)
	if err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("lock email confirmation user: %w", err)
	}
	if !eligible {
		return EmailConfirmation{}, false, nil
	}
	createdAt := store.now().UTC()
	confirmation, err := NewEmailConfirmation(createdAt)
	if err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("generate email confirmation: %w", err)
	}
	if err := transaction.InvalidateActive(ctx, parsedUserID, createdAt); err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("invalidate email confirmations: %w", err)
	}
	if err := transaction.Create(ctx, parsedUserID, confirmation, createdAt); err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("create email confirmation: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return EmailConfirmation{}, false, fmt.Errorf("commit email confirmation transaction: %w", err)
	}
	return confirmation, true, nil
}

func (store *Store) ConfirmEmailAndCreateSession(ctx context.Context, tokenHash, refreshTokenHash []byte, confirmedAt, refreshExpiresAt time.Time, beforeCommit func(auth.User) error) error {
	transaction, err := store.beginVerification(ctx)
	if err != nil {
		return fmt.Errorf("begin email confirmation verification: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	record, found, err := transaction.Lock(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("lock email confirmation: %w", err)
	}
	if !found || !validVerificationRecord(record, confirmedAt) {
		return ErrInvalidEmailConfirmation
	}
	confirmed, err := transaction.ConfirmUser(ctx, record.UserID, confirmedAt)
	if err != nil {
		return fmt.Errorf("confirm user email: %w", err)
	}
	if !confirmed {
		return ErrInvalidEmailConfirmation
	}
	consumed, err := transaction.Consume(ctx, record.ConfirmationID, confirmedAt)
	if err != nil {
		return fmt.Errorf("consume email confirmation: %w", err)
	}
	if !consumed {
		return ErrInvalidEmailConfirmation
	}
	if err := transaction.CreateTrial(ctx, record, NewTrialWindow(confirmedAt)); err != nil {
		return verificationWriteError("create establishment trial", err)
	}
	if err := transaction.CreateSession(ctx, record.UserID, refreshTokenHash, refreshExpiresAt); err != nil {
		return fmt.Errorf("create confirmed owner session: %w", err)
	}
	user := auth.User{
		ID: uuidString(record.UserID), EstablishmentID: uuidString(record.EstablishmentID), Email: record.Email,
		Role: record.Role, Active: record.Active, EmailConfirmed: true, CredentialVersion: record.CredentialVersion,
		PasswordChangeRequired: record.PasswordChangeRequired,
	}
	if err := beforeCommit(user); err != nil {
		return fmt.Errorf("issue confirmed owner authentication: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit email confirmation verification: %w", err)
	}
	return nil
}

func validVerificationRecord(record verificationRecord, now time.Time) bool {
	return record.Role == "owner" && record.Active && !record.EmailConfirmed && !record.Consumed && !record.Invalidated && record.ExpiresAt.After(now)
}

func verificationWriteError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.ConstraintName == "establishment_trials_owner_unique" || postgresError.ConstraintName == "establishment_trials_establishment_unique") {
		return ErrInvalidEmailConfirmation
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	bytes := value.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
