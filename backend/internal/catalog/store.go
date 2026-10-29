package catalog

import (
	"context"
	"errors"

	db "barbershop-app/backend/db/generated"

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
	parsedID, err := uuid.Parse(value)
	return pgtype.UUID{Bytes: parsedID, Valid: err == nil}, err
}

func offeringFromRow(row db.Service) Offering {
	return Offering{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, PriceCents: int64(row.PriceCents), DurationMinutes: int64(row.DurationMinutes), Active: row.Active}
}

func (store *PostgresStore) Create(ctx context.Context, establishmentID, name string, priceCents, durationMinutes int32) (Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return Offering{}, err
	}
	row, err := store.queries.CreateOffering(ctx, db.CreateOfferingParams{EstablishmentID: establishmentUUID, Name: name, PriceCents: priceCents, DurationMinutes: durationMinutes})
	return offeringFromRow(row), err
}

func (store *PostgresStore) List(ctx context.Context, establishmentID string, limit, offset int32) ([]Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return nil, err
	}
	rows, err := store.queries.ListOfferings(ctx, db.ListOfferingsParams{EstablishmentID: establishmentUUID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	items := make([]Offering, 0, len(rows))
	for _, row := range rows {
		items = append(items, offeringFromRow(row))
	}
	return items, nil
}

func (store *PostgresStore) Get(ctx context.Context, establishmentID, offeringID string) (Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return Offering{}, err
	}
	offeringUUID, err := parseUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	row, err := store.queries.GetOffering(ctx, db.GetOfferingParams{ID: offeringUUID, EstablishmentID: establishmentUUID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}

func (store *PostgresStore) Update(ctx context.Context, establishmentID, offeringID string, input UpdateInput) (Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return Offering{}, err
	}
	offeringUUID, err := parseUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	parameters := db.UpdateOfferingParams{ID: offeringUUID, EstablishmentID: establishmentUUID, SetName: input.Name != nil, SetPrice: input.PriceCents != nil, SetDuration: input.SetDuration}
	if input.Name != nil {
		parameters.Name = *input.Name
	}
	if input.PriceCents != nil {
		parameters.PriceCents = int32(*input.PriceCents)
	}
	if input.DurationMinutes != nil {
		parameters.DurationMinutes = int32(*input.DurationMinutes)
	}
	row, err := store.queries.UpdateOffering(ctx, parameters)
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}

func (store *PostgresStore) SetStatus(ctx context.Context, establishmentID, offeringID string, active bool) (Offering, error) {
	establishmentUUID, err := parseUUID(establishmentID)
	if err != nil {
		return Offering{}, err
	}
	offeringUUID, err := parseUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	row, err := store.queries.SetOfferingStatus(ctx, db.SetOfferingStatusParams{ID: offeringUUID, EstablishmentID: establishmentUUID, Active: active})
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}
