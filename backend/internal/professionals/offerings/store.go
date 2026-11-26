package offerings

import (
	"context"
	"errors"

	db "barbershop-app/backend/db/generated"
	"barbershop-app/backend/internal/catalog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ queries *db.Queries }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{queries: db.New(pool)}
}
func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	return pgtype.UUID{Bytes: parsed, Valid: err == nil}, err
}

func (store *PostgresStore) GetProfessional(ctx context.Context, establishmentID, professionalID string) (Professional, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return Professional{}, err
	}
	professionalUUID, err := parseUUID(professionalID)
	if err != nil {
		return Professional{}, err
	}
	row, err := store.queries.GetServicesProfessional(ctx, db.GetServicesProfessionalParams{UserID: professionalUUID, EstablishmentID: establishmentUUID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Professional{}, ErrNotFound
	}
	return Professional{ID: uuid.UUID(row.UserID.Bytes).String(), Name: row.Name, Active: row.Active}, err
}

func (store *PostgresStore) List(ctx context.Context, establishmentID, professionalID string, limit, offset int32) ([]Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return nil, err
	}
	professionalUUID, err := parseUUID(professionalID)
	if err != nil {
		return nil, err
	}
	rows, err := store.queries.ListProfessionalOfferings(ctx, db.ListProfessionalOfferingsParams{EstablishmentID: establishmentUUID, ProfessionalID: professionalUUID, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, err
	}
	items := make([]Offering, 0, len(rows))
	for _, row := range rows {
		items = append(items, Offering{Offering: catalog.Offering{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, PriceCents: int64(row.PriceCents), DurationMinutes: int64(row.DurationMinutes), Active: row.Active}, Associated: row.Associated})
	}
	return items, nil
}
