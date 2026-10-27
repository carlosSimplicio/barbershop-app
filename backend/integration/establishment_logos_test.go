//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestEstablishmentLogoMigrationPreservesExistingEstablishmentsAndProtectsRollback(t *testing.T) {
	ctx := context.Background()
	schemaName := "establishment_logo_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := integrationPool.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create migration test schema: %v", err)
	}
	connectionConfig := integrationPool.Config().Copy()
	if connectionConfig.ConnConfig.RuntimeParams == nil {
		connectionConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	connectionConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, connectionConfig)
	if err != nil {
		_, _ = integrationPool.Exec(ctx, "DROP SCHEMA "+schemaName+" CASCADE")
		t.Fatalf("open isolated migration pool: %v", err)
	}
	database := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() {
		_ = database.Close()
		pool.Close()
		_, _ = integrationPool.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
	})
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set Goose dialect: %v", err)
	}
	_, currentFile, _, _ := runtime.Caller(0)
	migrationDirectory := filepath.Join(filepath.Dir(currentFile), "..", "db", "migrations")
	if err := goose.UpToContext(ctx, database, migrationDirectory, 7); err != nil {
		t.Fatalf("apply baseline migrations: %v", err)
	}
	var establishmentID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO establishments (name, slug, time_zone, description, operational_data_started_at)
		VALUES ('Barbearia Legada', 'logo-legada', 'America/Sao_Paulo', 'Cadastro existente', statement_timestamp())
		RETURNING id
	`).Scan(&establishmentID); err != nil {
		t.Fatalf("create establishment before logo migration: %v", err)
	}
	if err := goose.UpContext(ctx, database, migrationDirectory); err != nil {
		t.Fatalf("apply logo migration: %v", err)
	}
	var logoID sql.NullString
	var description string
	if err := pool.QueryRow(ctx, "SELECT description, logo_id::text FROM establishments WHERE id = $1", establishmentID).Scan(&description, &logoID); err != nil {
		t.Fatalf("read establishment after logo migration: %v", err)
	}
	if description != "Cadastro existente" || logoID.Valid {
		t.Fatalf("logo migration changed existing profile: description=%q logo_id=%+v", description, logoID)
	}
	if _, err := db.New(pool).CreateEstablishmentLogo(ctx, logoParams(establishmentID, "logos/persisted")); err != nil {
		t.Fatalf("create metadata before rollback check: %v", err)
	}
	if err := goose.DownToContext(ctx, database, migrationDirectory, 7); err == nil {
		t.Fatal("rollback removed populated logo metadata instead of refusing")
	}
	var logoCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM establishment_logos WHERE establishment_id = $1", establishmentID).Scan(&logoCount); err != nil || logoCount != 1 {
		t.Fatalf("metadata after refused rollback = %d, err=%v; want 1", logoCount, err)
	}
}

func TestEstablishmentLogoAssociationIsTenantScopedAndTransactional(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	firstEstablishmentID := createEstablishmentLogoTestEstablishment(t, "logo-first-tenant")
	secondEstablishmentID := createEstablishmentLogoTestEstablishment(t, "logo-second-tenant")
	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin logo association transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback(ctx) })
	transactionQueries := db.New(transaction)
	firstLogo, err := transactionQueries.CreateEstablishmentLogo(ctx, logoParams(firstEstablishmentID, "logos/first"))
	if err != nil {
		t.Fatalf("create first logo metadata: %v", err)
	}
	firstLogoID := firstLogo.ID
	if rows, err := transactionQueries.SetCurrentEstablishmentLogo(ctx, db.SetCurrentEstablishmentLogoParams{
		EstablishmentID: establishmentUUID(firstEstablishmentID), LogoID: firstLogoID,
	}); err != nil || rows != 1 {
		t.Fatalf("associate first logo: %v", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("commit first logo association: %v", err)
	}

	currentLogo, err := db.New(integrationPool).GetCurrentEstablishmentLogo(ctx, establishmentUUID(firstEstablishmentID))
	if err != nil || currentLogo.ID != firstLogoID || currentLogo.EstablishmentID != establishmentUUID(firstEstablishmentID) || currentLogo.ObjectKey != "logos/first" {
		t.Fatalf("current logo = %+v, err=%v; want metadata for %s", currentLogo, err, firstLogoID)
	}
	if rows, err := db.New(integrationPool).SetCurrentEstablishmentLogo(ctx, db.SetCurrentEstablishmentLogoParams{
		EstablishmentID: establishmentUUID(firstEstablishmentID), LogoID: pgtype.UUID{},
	}); err != nil || rows != 0 {
		t.Fatalf("attempt to clear current logo: rows=%d err=%v; want no update", rows, err)
	}
	currentLogo, err = db.New(integrationPool).GetCurrentEstablishmentLogo(ctx, establishmentUUID(firstEstablishmentID))
	if err != nil || currentLogo.ID != firstLogoID {
		t.Fatalf("logo after rejected clear = %+v, err=%v; want original %s", currentLogo, err, firstLogoID)
	}

	secondLogo := createEstablishmentLogoTestMetadata(t, firstEstablishmentID, "logos/second")
	replacement, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin logo replacement transaction: %v", err)
	}
	if rows, err := db.New(replacement).SetCurrentEstablishmentLogo(ctx, db.SetCurrentEstablishmentLogoParams{
		EstablishmentID: establishmentUUID(firstEstablishmentID), LogoID: secondLogo,
	}); err != nil || rows != 1 {
		_ = replacement.Rollback(ctx)
		t.Fatalf("replace logo within transaction: %v", err)
	}
	if err := replacement.Rollback(ctx); err != nil {
		t.Fatalf("roll back logo replacement: %v", err)
	}
	currentLogo, err = db.New(integrationPool).GetCurrentEstablishmentLogo(ctx, establishmentUUID(firstEstablishmentID))
	if err != nil || currentLogo.ID != firstLogoID {
		t.Fatalf("logo after rollback = %+v, err=%v; want original %s", currentLogo, err, firstLogoID)
	}

	_, err = db.New(integrationPool).SetCurrentEstablishmentLogo(ctx, db.SetCurrentEstablishmentLogoParams{
		EstablishmentID: establishmentUUID(secondEstablishmentID), LogoID: firstLogoID,
	})
	if err == nil {
		t.Fatal("database accepted a logo association across establishments")
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23503" {
		t.Fatalf("cross-tenant logo association error = %v, want foreign key violation", err)
	}
	_, err = db.New(integrationPool).GetCurrentEstablishmentLogo(ctx, establishmentUUID(secondEstablishmentID))
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("second establishment current logo query error = %v, want no rows", err)
	}
}

func createEstablishmentLogoTestEstablishment(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := integrationPool.QueryRow(context.Background(), `
		INSERT INTO establishments (name, slug, time_zone)
		VALUES ('Barbearia Logo', $1, 'America/Sao_Paulo')
		RETURNING id
	`, slug).Scan(&id); err != nil {
		t.Fatalf("create establishment %q: %v", slug, err)
	}
	return id
}

func createEstablishmentLogoTestMetadata(t *testing.T, establishmentID uuid.UUID, objectKey string) pgtype.UUID {
	t.Helper()
	logo, err := db.New(integrationPool).CreateEstablishmentLogo(context.Background(), logoParams(establishmentID, objectKey))
	if err != nil {
		t.Fatalf("create logo metadata for establishment %s: %v", establishmentID, err)
	}
	return logo.ID
}

func logoParams(establishmentID uuid.UUID, objectKey string) db.CreateEstablishmentLogoParams {
	return db.CreateEstablishmentLogoParams{
		EstablishmentID: establishmentUUID(establishmentID), ObjectKey: objectKey,
		ContentType: "image/png", ByteSize: 100, Width: 1, Height: 1,
		ContentSha256: make([]byte, 32),
	}
}
