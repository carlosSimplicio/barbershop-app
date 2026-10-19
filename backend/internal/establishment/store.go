package establishment

import (
	"context"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type establishmentQueries interface {
	GetEstablishmentWithTrial(context.Context, pgtype.UUID) (db.GetEstablishmentWithTrialRow, error)
}

type PostgresStore struct {
	queries establishmentQueries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{queries: db.New(pool)}
}

func (store *PostgresStore) GetByID(ctx context.Context, establishmentID string) (Establishment, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return Establishment{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	row, err := store.queries.GetEstablishmentWithTrial(ctx, pgtype.UUID{Bytes: parsedID, Valid: true})
	if err != nil {
		return Establishment{}, fmt.Errorf("query establishment: %w", err)
	}

	result := Establishment{
		ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Slug: row.Slug, TimeZone: row.TimeZone,
	}
	if row.StartsAt.Valid != row.EndsAt.Valid {
		return Establishment{}, fmt.Errorf("query establishment: inconsistent trial limits")
	}
	if row.StartsAt.Valid {
		result.Trial = &Trial{StartsAt: row.StartsAt.Time, EndsAt: row.EndsAt.Time}
	}
	return result, nil
}
