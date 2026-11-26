//go:build integration

package integration_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestWeeklyScheduleMigrationPreservesExistingProfessionals(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := "weekly_schedule_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := integrationPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := integrationPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})

	config := integrationPool.Config()
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	database := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = database.Close() })

	_, currentFile, _, _ := runtime.Caller(0)
	directory := filepath.Join(filepath.Dir(currentFile), "..", "db", "migrations")
	if err := goose.UpToContext(ctx, database, directory, 10); err != nil {
		t.Fatal(err)
	}
	tenant, otherTenant := uuid.New(), uuid.New()
	professional, colleague := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{tenant, otherTenant} {
		if _, err := pool.Exec(ctx, `INSERT INTO establishments (id, name, slug, time_zone) VALUES ($1, 'Salão fictício', $2, 'America/Sao_Paulo')`, id, id.String()); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []uuid.UUID{professional, colleague} {
		if _, err := pool.Exec(ctx, `INSERT INTO users (id, establishment_id, email, normalized_email, role) VALUES ($1, $2, $3, $3, 'professional')`, id, tenant, id.String()+"@example.test"); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO professionals (user_id, establishment_id, name) VALUES ($1, $2, 'Profissional fictício')`, id, tenant); err != nil {
			t.Fatal(err)
		}
	}
	var serviceID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO services (establishment_id, name, price_cents, duration_minutes)
		VALUES ($1, 'Serviço fictício', 1250, 30) RETURNING id`, tenant).Scan(&serviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO professional_services (establishment_id, professional_id, service_id) VALUES ($1, $2, $3)`, tenant, professional, serviceID); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, database, directory, 11); err != nil {
		t.Fatalf("apply weekly schedule migration: %v", err)
	}

	const insertPeriod = `INSERT INTO professional_work_periods
		(establishment_id, professional_id, weekday, start_minute, end_minute)
		VALUES ($1, $2, $3, $4, $5)`
	var name string
	var associationActive bool
	if err := pool.QueryRow(ctx, `SELECT p.name, association.active
		FROM professionals AS p JOIN professional_services AS association
			ON (p.user_id, p.establishment_id) = (association.professional_id, association.establishment_id)
		WHERE p.user_id = $1 AND p.establishment_id = $2 AND association.service_id = $3`, professional, tenant, serviceID).
		Scan(&name, &associationActive); err != nil || name != "Profissional fictício" || !associationActive {
		t.Fatalf("existing registration and association were not preserved: %q, %v, %v", name, associationActive, err)
	}

	list := func(establishmentID, professionalID uuid.UUID) []db.ListProfessionalWorkPeriodsRow {
		t.Helper()
		periods, err := db.New(pool).ListProfessionalWorkPeriods(ctx, db.ListProfessionalWorkPeriodsParams{
			EstablishmentID: pgtype.UUID{Bytes: establishmentID, Valid: true},
			ProfessionalID:  pgtype.UUID{Bytes: professionalID, Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
		return periods
	}
	if periods := list(tenant, professional); len(periods) != 0 {
		t.Fatalf("existing professional received default periods: %+v", periods)
	}

	_, err = pool.Exec(ctx, insertPeriod, otherTenant, professional, 1, 480, 720)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.ConstraintName != "professional_work_periods_professional_fk" {
		t.Fatalf("cross-tenant period error = %v, want professional foreign key violation", err)
	}

	for _, period := range [][3]int{{7, 720, 1440}, {1, 780, 1080}, {1, 480, 720}} {
		if _, err := pool.Exec(ctx, insertPeriod, tenant, professional, period[0], period[1], period[2]); err != nil {
			t.Fatal(err)
		}
	}
	periods := list(tenant, professional)
	if len(periods) != 3 || periods[0].Weekday != 1 || periods[0].StartMinute != 480 ||
		periods[1].Weekday != 1 || periods[1].StartMinute != 780 ||
		periods[2].Weekday != 7 || periods[2].EndMinute != 1440 {
		t.Fatalf("periods are incomplete or out of order: %+v", periods)
	}
	if len(list(otherTenant, professional)) != 0 || len(list(tenant, colleague)) != 0 {
		t.Fatal("periods leaked into another tenant or professional")
	}

	if err := goose.DownToContext(ctx, database, directory, 10); err == nil {
		t.Fatal("rollback succeeded with configured periods")
	}
	if len(list(tenant, professional)) != len(periods) {
		t.Fatal("failed rollback discarded configured periods")
	}
}
