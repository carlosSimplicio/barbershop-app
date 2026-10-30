package professionals

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (store *PostgresStore) Create(ctx context.Context, creation Creation) (string, error) {
	establishmentID, err := uuid.Parse(creation.EstablishmentID)
	if err != nil {
		return "", fmt.Errorf("parse establishment ID: %w", err)
	}
	tenant := pgtype.UUID{Bytes: establishmentID, Valid: true}
	at := pgtype.Timestamptz{Time: creation.CreatedAt, Valid: true}

	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin professional transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	queries := db.New(transaction)

	userID, err := queries.CreateProfessionalUser(ctx, db.CreateProfessionalUserParams{
		EstablishmentID: tenant, Email: creation.Email, NormalizedEmail: creation.NormalizedEmail, CreatedAt: at,
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.ConstraintName == "users_normalized_email_unique" {
			return "", ErrEmailUnavailable
		}
		return "", fmt.Errorf("create professional user: %w", err)
	}
	if err := queries.CreateProfessionalCredential(ctx, db.CreateProfessionalCredentialParams{
		UserID: userID, PasswordHash: creation.PasswordHash, CreatedAt: at,
	}); err != nil {
		return "", fmt.Errorf("create professional credential: %w", err)
	}
	if err := queries.CreateProfessionalProfile(ctx, db.CreateProfessionalProfileParams{
		UserID: userID, EstablishmentID: tenant, Name: creation.Name, CreatedAt: at,
	}); err != nil {
		return "", fmt.Errorf("create professional profile: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit professional transaction: %w", err)
	}
	return uuid.UUID(userID.Bytes).String(), nil
}

func (store *PostgresStore) List(ctx context.Context, establishmentID string, limit, offset int32) ([]Professional, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}
	rows, err := db.New(store.pool).ListProfessionalsByEstablishment(ctx, db.ListProfessionalsByEstablishmentParams{
		EstablishmentID: pgtype.UUID{Bytes: parsedID, Valid: true}, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("query professionals: %w", err)
	}
	items := make([]Professional, 0, len(rows))
	for _, row := range rows {
		items = append(items, Professional{
			ID: uuid.UUID(row.UserID.Bytes).String(), Name: row.Name, Email: row.Email, Active: row.Active,
		})
	}
	return items, nil
}

func (store *PostgresStore) Get(ctx context.Context, establishmentID, professionalID string) (Professional, error) {
	tenant, err := uuid.Parse(establishmentID)
	if err != nil {
		return Professional{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	professionalUUID, err := uuid.Parse(professionalID)
	if err != nil {
		return Professional{}, ErrNotFound
	}
	row, err := db.New(store.pool).GetProfessional(ctx, db.GetProfessionalParams{
		UserID: pgtype.UUID{Bytes: professionalUUID, Valid: true}, EstablishmentID: pgtype.UUID{Bytes: tenant, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	}
	if err != nil {
		return Professional{}, fmt.Errorf("get professional: %w", err)
	}
	return Professional{ID: uuid.UUID(row.UserID.Bytes).String(), Name: row.Name, Email: row.Email, Active: row.Active}, nil
}

func (store *PostgresStore) UpdateName(ctx context.Context, establishmentID, professionalID, name string, at time.Time) (Professional, error) {
	tenant, err := uuid.Parse(establishmentID)
	if err != nil {
		return Professional{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	professionalUUID, err := uuid.Parse(professionalID)
	if err != nil {
		return Professional{}, ErrNotFound
	}
	// Lock the user before the profile so RETURNING cannot combine a new name
	// with an active state from before a concurrent state transition.
	row, err := db.New(store.pool).UpdateProfessionalName(ctx, db.UpdateProfessionalNameParams{
		ID: pgtype.UUID{Bytes: professionalUUID, Valid: true}, EstablishmentID: pgtype.UUID{Bytes: tenant, Valid: true}, Name: name, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	}
	if err != nil {
		return Professional{}, fmt.Errorf("update professional name: %w", err)
	}
	return Professional{ID: uuid.UUID(row.UserID.Bytes).String(), Name: row.Name, Email: row.Email, Active: row.Active}, nil
}

func (store *PostgresStore) Deactivate(ctx context.Context, establishmentID, professionalID string, at time.Time) (Professional, error) {
	tenant, err := uuid.Parse(establishmentID)
	if err != nil {
		return Professional{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	professionalUUID, err := uuid.Parse(professionalID)
	if err != nil {
		return Professional{}, ErrNotFound
	}
	tenantID := pgtype.UUID{Bytes: tenant, Valid: true}
	userID := pgtype.UUID{Bytes: professionalUUID, Valid: true}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return Professional{}, fmt.Errorf("begin deactivation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	queries := db.New(transaction)
	// Serialize state transitions with login and password changes before reading
	// the profile or writing credentials and session revocations.
	if _, err = queries.LockProfessionalUser(ctx, db.LockProfessionalUserParams{ID: userID, EstablishmentID: tenantID}); errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	} else if err != nil {
		return Professional{}, fmt.Errorf("lock professional: %w", err)
	}
	row, err := queries.GetProfessional(ctx, db.GetProfessionalParams{UserID: userID, EstablishmentID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	}
	if err != nil {
		return Professional{}, fmt.Errorf("read professional for deactivation: %w", err)
	}
	if row.Active {
		if err = queries.DeactivateProfessionalUser(ctx, db.DeactivateProfessionalUserParams{ID: userID, EstablishmentID: tenantID, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true}}); err != nil {
			return Professional{}, fmt.Errorf("deactivate professional: %w", err)
		}
		if err = queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{UserID: userID, UpdatedAt: pgtype.Timestamptz{Time: at, Valid: true}}); err != nil {
			return Professional{}, fmt.Errorf("revoke professional sessions: %w", err)
		}
	}
	if err = transaction.Commit(ctx); err != nil {
		return Professional{}, fmt.Errorf("commit deactivation: %w", err)
	}
	return Professional{ID: professionalID, Name: row.Name, Email: row.Email, Active: false}, nil
}

func (store *PostgresStore) Reactivate(ctx context.Context, establishmentID, professionalID, passwordHash string, at time.Time) (Professional, error) {
	tenant, err := uuid.Parse(establishmentID)
	if err != nil {
		return Professional{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	professionalUUID, err := uuid.Parse(professionalID)
	if err != nil {
		return Professional{}, ErrNotFound
	}
	tenantID := pgtype.UUID{Bytes: tenant, Valid: true}
	userID := pgtype.UUID{Bytes: professionalUUID, Valid: true}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return Professional{}, fmt.Errorf("begin reactivation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	queries := db.New(transaction)
	if _, err = queries.LockProfessionalUser(ctx, db.LockProfessionalUserParams{ID: userID, EstablishmentID: tenantID}); errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	} else if err != nil {
		return Professional{}, fmt.Errorf("lock professional: %w", err)
	}
	row, err := queries.GetProfessional(ctx, db.GetProfessionalParams{UserID: userID, EstablishmentID: tenantID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	}
	if err != nil {
		return Professional{}, fmt.Errorf("read professional for reactivation: %w", err)
	}
	if row.Active {
		return Professional{}, ErrAlreadyActive
	}
	updatedAt := pgtype.Timestamptz{Time: at, Valid: true}
	if err = queries.ReactivateProfessionalUser(ctx, db.ReactivateProfessionalUserParams{ID: userID, EstablishmentID: tenantID, UpdatedAt: updatedAt}); err != nil {
		return Professional{}, fmt.Errorf("reactivate professional: %w", err)
	}
	affected, err := queries.ReplaceProfessionalCredential(ctx, db.ReplaceProfessionalCredentialParams{UserID: userID, PasswordHash: passwordHash, UpdatedAt: updatedAt})
	if err != nil {
		return Professional{}, fmt.Errorf("replace professional credential: %w", err)
	}
	if affected != 1 {
		return Professional{}, errors.New("professional credential missing")
	}
	if err = queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{UserID: userID, UpdatedAt: updatedAt}); err != nil {
		return Professional{}, fmt.Errorf("revoke professional sessions: %w", err)
	}
	if err = transaction.Commit(ctx); err != nil {
		return Professional{}, fmt.Errorf("commit reactivation: %w", err)
	}
	return Professional{ID: professionalID, Name: row.Name, Email: row.Email, Active: true}, nil
}
