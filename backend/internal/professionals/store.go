package professionals

import (
	"context"
	"errors"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
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

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin professional transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)

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
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit professional transaction: %w", err)
	}
	return uuid.UUID(userID.Bytes).String(), nil
}
