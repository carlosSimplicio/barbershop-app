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
func tenantUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	return pgtype.UUID{Bytes: parsed, Valid: err == nil}, err
}
func offeringFromRow(row db.Service) Offering {
	return Offering{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, PriceCents: int64(row.PriceCents), DurationMinutes: int64(row.DurationMinutes), Active: row.Active}
}
func (s *PostgresStore) Create(ctx context.Context, tenant, name string, price, duration int32) (Offering, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Offering{}, err
	}
	row, err := s.queries.CreateOffering(ctx, db.CreateOfferingParams{EstablishmentID: id, Name: name, PriceCents: price, DurationMinutes: duration})
	return offeringFromRow(row), err
}

func (s *PostgresStore) List(ctx context.Context, tenant string, limit, offset int32) ([]Offering, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListOfferings(ctx, db.ListOfferingsParams{EstablishmentID: id, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	items := make([]Offering, 0, len(rows))
	for _, row := range rows {
		items = append(items, offeringFromRow(row))
	}
	return items, nil
}

func (s *PostgresStore) Get(ctx context.Context, tenant, offeringID string) (Offering, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Offering{}, err
	}
	target, err := tenantUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	row, err := s.queries.GetOffering(ctx, db.GetOfferingParams{ID: target, EstablishmentID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}

func (s *PostgresStore) Update(ctx context.Context, tenant, offeringID string, in UpdateInput) (Offering, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Offering{}, err
	}
	target, err := tenantUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	params := db.UpdateOfferingParams{ID: target, EstablishmentID: id, SetName: in.Name != nil, SetPrice: in.PriceCents != nil, SetDuration: in.SetDuration}
	if in.Name != nil {
		params.Name = *in.Name
	}
	if in.PriceCents != nil {
		params.PriceCents = int32(*in.PriceCents)
	}
	if in.DurationMinutes != nil {
		params.DurationMinutes = int32(*in.DurationMinutes)
	}
	row, err := s.queries.UpdateOffering(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}

func (s *PostgresStore) SetStatus(ctx context.Context, tenant, offeringID string, active bool) (Offering, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Offering{}, err
	}
	target, err := tenantUUID(offeringID)
	if err != nil {
		return Offering{}, err
	}
	row, err := s.queries.SetOfferingStatus(ctx, db.SetOfferingStatusParams{ID: target, EstablishmentID: id, Active: active})
	if errors.Is(err, pgx.ErrNoRows) {
		return Offering{}, ErrNotFound
	}
	return offeringFromRow(row), err
}
