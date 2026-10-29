package services

import (
	db "barbershop-app/backend/db/generated"
	"context"
	"errors"
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
func recordFromRow(row db.Service) Record {
	return Record{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, PriceCents: int64(row.PriceCents), DurationMinutes: int64(row.DurationMinutes), Active: row.Active}
}
func (s *PostgresStore) Create(ctx context.Context, tenant, name string, price, duration int32) (Record, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Record{}, err
	}
	row, err := s.queries.CreateService(ctx, db.CreateServiceParams{EstablishmentID: id, Name: name, PriceCents: price, DurationMinutes: duration})
	return recordFromRow(row), err
}

func (s *PostgresStore) List(ctx context.Context, tenant string, limit, offset int32) ([]Record, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListServices(ctx, db.ListServicesParams{EstablishmentID: id, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	items := make([]Record, 0, len(rows))
	for _, row := range rows {
		items = append(items, recordFromRow(row))
	}
	return items, nil
}

func (s *PostgresStore) Get(ctx context.Context, tenant, serviceID string) (Record, error) {
	id, err := tenantUUID(tenant)
	if err != nil {
		return Record{}, err
	}
	target, err := tenantUUID(serviceID)
	if err != nil {
		return Record{}, err
	}
	row, err := s.queries.GetService(ctx, db.GetServiceParams{ID: target, EstablishmentID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	return recordFromRow(row), err
}
