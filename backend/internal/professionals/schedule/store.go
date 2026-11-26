package schedule

import (
	"context"
	"errors"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	queries *db.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{queries: db.New(pool)}
}

func (store *PostgresStore) Read(ctx context.Context, establishmentID string, professionalID uuid.UUID) (Week, error) {
	tenant, err := uuid.Parse(establishmentID)
	if err != nil {
		return Week{}, err
	}
	tenantParam := pgtype.UUID{Bytes: tenant, Valid: true}
	professionalParam := pgtype.UUID{Bytes: professionalID, Valid: true}
	timeZone, err := store.queries.GetProfessionalScheduleTimeZone(ctx, db.GetProfessionalScheduleTimeZoneParams{
		EstablishmentID: tenantParam,
		UserID:          professionalParam,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Week{}, ErrNotFound
	}
	if err != nil {
		return Week{}, err
	}
	rows, err := store.queries.ListProfessionalWorkPeriods(ctx, db.ListProfessionalWorkPeriodsParams{
		EstablishmentID: tenantParam,
		ProfessionalID:  professionalParam,
	})
	if err != nil {
		return Week{}, err
	}
	periods := make([]Period, 0, len(rows))
	for _, row := range rows {
		periods = append(periods, Period(row))
	}
	return Week{TimeZone: timeZone, Periods: periods}, nil
}
