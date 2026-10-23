package entitlements

import (
	"context"
	"errors"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type trialQueries interface {
	GetTrialWindowByEstablishment(context.Context, pgtype.UUID) (db.GetTrialWindowByEstablishmentRow, error)
}

type PostgresStore struct {
	queries trialQueries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{queries: db.New(pool)}
}

func (store *PostgresStore) GetTrialWindow(ctx context.Context, establishmentID string) (*TrialWindow, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return nil, fmt.Errorf("parse establishment ID: %w", err)
	}
	row, err := store.queries.GetTrialWindowByEstablishment(ctx, pgtype.UUID{Bytes: parsedID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query trial window: %w", err)
	}
	return &TrialWindow{StartsAt: row.StartsAt.Time, EndsAt: row.EndsAt.Time}, nil
}
