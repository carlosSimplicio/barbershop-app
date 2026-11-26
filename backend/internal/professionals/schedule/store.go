package schedule

import (
	"context"
	"errors"
	"slices"

	db "barbershop-app/backend/db/generated"
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: db.New(pool)}
}

func (store *PostgresStore) Read(ctx context.Context, establishmentID string, professionalID uuid.UUID) (Week, error) {
	tenant, err := parseTenant(establishmentID)
	if err != nil {
		return Week{}, err
	}
	rows, err := store.queries.GetProfessionalSchedule(ctx, db.GetProfessionalScheduleParams{
		EstablishmentID: tenant,
		UserID:          pgtype.UUID{Bytes: professionalID, Valid: true},
	})
	if err != nil {
		return Week{}, err
	}
	if len(rows) == 0 {
		return Week{}, ErrNotFound
	}
	// The outer join returns weekday zero for a valid professional with an empty week.
	periods := make([]Period, 0, len(rows))
	for _, row := range rows {
		if row.Weekday != 0 {
			periods = append(periods, Period{row.Weekday, row.StartMinute, row.EndMinute})
		}
	}
	return Week{TimeZone: rows[0].TimeZone, Periods: periods}, nil
}

func (store *PostgresStore) CheckTarget(ctx context.Context, establishmentID string, professionalID uuid.UUID) error {
	tenant, err := parseTenant(establishmentID)
	if err != nil {
		return err
	}
	_, err = store.queries.GetProfessional(ctx, db.GetProfessionalParams{
		EstablishmentID: tenant,
		UserID:          pgtype.UUID{Bytes: professionalID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (store *PostgresStore) Replace(ctx context.Context, principal auth.Principal, professionalID uuid.UUID, periods []Period, revalidateWrite func(context.Context, entitlements.Store) error) (Week, error) {
	tenant, err := parseTenant(principal.EstablishmentID)
	if err != nil {
		return Week{}, err
	}
	professional := pgtype.UUID{Bytes: professionalID, Valid: true}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return Week{}, err
	}
	defer transaction.Rollback(ctx)
	queries := store.queries.WithTx(transaction)

	// The user lock orders full-week replacements with one another and with
	// deactivation and reactivation. The version must come from authentication.
	user, err := queries.LockProfessionalUser(ctx, db.LockProfessionalUserParams{EstablishmentID: tenant, ID: professional})
	if errors.Is(err, pgx.ErrNoRows) {
		return Week{}, ErrNotFound
	}
	if err != nil {
		return Week{}, err
	}
	if principal.Role == "professional" && (!user.Active || user.CredentialVersion != principal.CredentialVersion) {
		return Week{}, ErrStalePrincipal
	}
	current, err := queries.ListProfessionalWorkPeriods(ctx, db.ListProfessionalWorkPeriodsParams{EstablishmentID: tenant, ProfessionalID: professional})
	if err != nil {
		return Week{}, err
	}
	unchanged := slices.EqualFunc(current, periods, func(left db.ListProfessionalWorkPeriodsRow, right Period) bool {
		return left.Weekday == right.Weekday && left.StartMinute == right.StartMinute && left.EndMinute == right.EndMinute
	})
	if !unchanged {
		if err := queries.DeleteProfessionalWorkPeriods(ctx, db.DeleteProfessionalWorkPeriodsParams{EstablishmentID: tenant, ProfessionalID: professional}); err != nil {
			return Week{}, err
		}
		if len(periods) > 0 {
			if err := insertPeriods(ctx, queries, tenant, professional, periods); err != nil {
				return Week{}, err
			}
		}
	}
	// Read the zone after the replacement so a settings update can finish while
	// periods are written. The shared lock keeps that zone current until commit.
	timeZone, err := queries.LockProfessionalScheduleTimeZone(ctx, db.LockProfessionalScheduleTimeZoneParams{
		EstablishmentID: tenant,
		UserID:          professional,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Week{}, ErrNotFound
	}
	if err != nil {
		return Week{}, err
	}
	// Recheck after all lock waits using the existing transaction connection.
	if err := revalidateWrite(ctx, entitlements.NewPostgresStore(transaction)); err != nil {
		return Week{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return Week{}, err
	}
	return Week{TimeZone: timeZone, Periods: periods}, nil
}

func parseTenant(establishmentID string) (pgtype.UUID, error) {
	tenant, err := uuid.Parse(establishmentID)
	return pgtype.UUID{Bytes: tenant, Valid: err == nil}, err
}

func insertPeriods(ctx context.Context, queries *db.Queries, tenant, professional pgtype.UUID, periods []Period) error {
	input := db.InsertProfessionalWorkPeriodsParams{
		EstablishmentID: tenant,
		ProfessionalID:  professional,
		Weekdays:        make([]int16, 0, len(periods)),
		StartMinutes:    make([]int16, 0, len(periods)),
		EndMinutes:      make([]int16, 0, len(periods)),
	}
	for _, period := range periods {
		input.Weekdays = append(input.Weekdays, period.Weekday)
		input.StartMinutes = append(input.StartMinutes, period.StartMinute)
		input.EndMinutes = append(input.EndMinutes, period.EndMinute)
	}
	return queries.InsertProfessionalWorkPeriods(ctx, input)
}
