//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestEstablishmentSettingsMigrationPreservesExistingEstablishment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaName := "settings_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	if err := goose.UpToContext(ctx, database, migrationDirectory, 5); err != nil {
		t.Fatalf("apply baseline migrations: %v", err)
	}
	const (
		legacyName     = "Barbearia legado"
		legacySlug     = "barbearia-legado"
		legacyTimeZone = "America/Sao_Paulo"
	)
	var establishmentID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO establishments (name, slug, time_zone)
		VALUES ($1, $2, $3)
		RETURNING id
	`, legacyName, legacySlug, legacyTimeZone).Scan(&establishmentID)
	if err != nil {
		t.Fatalf("insert establishment before settings migration: %v", err)
	}
	if err := goose.UpContext(ctx, database, migrationDirectory); err != nil {
		t.Fatalf("apply settings migration: %v", err)
	}

	var name, slug, timeZone sql.NullString
	var description, address, contactPhone sql.NullString
	var minimumNotice, bookingHorizon, changeDeadline sql.NullInt64
	err = pool.QueryRow(ctx, `
		SELECT name, slug, time_zone, description, address, contact_phone,
			minimum_booking_notice_minutes, booking_horizon_days, customer_change_deadline_minutes
		FROM establishments
		WHERE id = $1
	`, establishmentID).Scan(
		&name, &slug, &timeZone, &description, &address, &contactPhone,
		&minimumNotice, &bookingHorizon, &changeDeadline,
	)
	if err != nil {
		t.Fatalf("read establishment after settings migration: %v", err)
	}
	if !name.Valid || name.String != legacyName || !slug.Valid || slug.String != legacySlug || !timeZone.Valid || timeZone.String != legacyTimeZone {
		t.Fatalf("migration changed existing identity: name=%+v slug=%+v time_zone=%+v", name, slug, timeZone)
	}
	if description.Valid || address.Valid || contactPhone.Valid || minimumNotice.Valid || bookingHorizon.Valid || changeDeadline.Valid {
		t.Fatalf("new optional settings should start unset: description=%+v address=%+v phone=%+v notice=%+v horizon=%+v deadline=%+v",
			description, address, contactPhone, minimumNotice, bookingHorizon, changeDeadline)
	}
	var socialLinkCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM establishment_social_links WHERE establishment_id = $1", establishmentID).Scan(&socialLinkCount); err != nil {
		t.Fatalf("count social links after settings migration: %v", err)
	}
	if socialLinkCount != 0 {
		t.Fatalf("existing establishment has %d unexpected social links", socialLinkCount)
	}
	if _, err := pool.Exec(ctx, "UPDATE establishments SET description = 'Descrição salva' WHERE id = $1", establishmentID); err != nil {
		t.Fatalf("save settings before rollback check: %v", err)
	}
	if err := goose.DownToContext(ctx, database, migrationDirectory, 5); err == nil {
		t.Fatal("rollback removed saved settings instead of refusing")
	}
	var preservedDescription string
	if err := pool.QueryRow(ctx, "SELECT description FROM establishments WHERE id = $1", establishmentID).Scan(&preservedDescription); err != nil {
		t.Fatalf("read settings after refused rollback: %v", err)
	}
	if preservedDescription != "Descrição salva" {
		t.Fatalf("rollback changed saved settings: %q", preservedDescription)
	}
}

func TestEstablishmentSettingsQueriesPersistValuesAndScopeLinksToTenant(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	createEstablishment := func(name, slug string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := integrationPool.QueryRow(ctx, `
			INSERT INTO establishments (name, slug, time_zone)
			VALUES ($1, $2, 'America/Sao_Paulo')
			RETURNING id
		`, name, slug).Scan(&id); err != nil {
			t.Fatalf("create establishment %q: %v", slug, err)
		}
		return id
	}
	firstTenant := createEstablishment("Primeira", "primeira-settings")
	secondTenant := createEstablishment("Segunda", "segunda-settings")
	queries := db.New(integrationPool)

	rowsAffected, err := queries.UpdateEstablishmentPublicSettings(ctx, db.UpdateEstablishmentPublicSettingsParams{
		Description:                   pgtype.Text{String: "Cortes e cuidados", Valid: true},
		Address:                       pgtype.Text{String: "Rua Um, 10", Valid: true},
		ContactPhone:                  pgtype.Text{String: "+5511999990000", Valid: true},
		MinimumBookingNoticeMinutes:   pgtype.Int4{Int32: 30, Valid: true},
		BookingHorizonDays:            pgtype.Int4{Int32: 45, Valid: true},
		CustomerChangeDeadlineMinutes: pgtype.Int4{Int32: 120, Valid: true},
		EstablishmentID:               pgtype.UUID{Bytes: firstTenant, Valid: true},
	})
	if err != nil {
		t.Fatalf("save first establishment settings: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("updated rows = %d, want 1", rowsAffected)
	}
	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin social link transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback(ctx) })
	transactionQueries := db.New(transaction)
	if err := transactionQueries.CreateEstablishmentSocialLink(ctx, db.CreateEstablishmentSocialLinkParams{
		EstablishmentID: pgtype.UUID{Bytes: firstTenant, Valid: true}, Position: 1, Label: "Instagram", Url: "https://instagram.com/primeira",
	}); err != nil {
		t.Fatalf("save second-position social link: %v", err)
	}
	if err := transactionQueries.CreateEstablishmentSocialLink(ctx, db.CreateEstablishmentSocialLinkParams{
		EstablishmentID: pgtype.UUID{Bytes: firstTenant, Valid: true}, Position: 0, Label: "Site", Url: "https://primeira.example.test",
	}); err != nil {
		t.Fatalf("save first-position social link: %v", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("commit social link transaction: %v", err)
	}

	stored, err := queries.GetEstablishmentPublicSettings(ctx, pgtype.UUID{Bytes: firstTenant, Valid: true})
	if err != nil {
		t.Fatalf("read first establishment settings: %v", err)
	}
	if !stored.Description.Valid || stored.Description.String != "Cortes e cuidados" ||
		!stored.Address.Valid || stored.Address.String != "Rua Um, 10" ||
		!stored.ContactPhone.Valid || stored.ContactPhone.String != "+5511999990000" ||
		!stored.MinimumBookingNoticeMinutes.Valid || stored.MinimumBookingNoticeMinutes.Int32 != 30 ||
		!stored.BookingHorizonDays.Valid || stored.BookingHorizonDays.Int32 != 45 ||
		!stored.CustomerChangeDeadlineMinutes.Valid || stored.CustomerChangeDeadlineMinutes.Int32 != 120 {
		t.Fatalf("stored settings do not match: %+v", stored)
	}
	links, err := queries.ListEstablishmentSocialLinks(ctx, pgtype.UUID{Bytes: firstTenant, Valid: true})
	if err != nil {
		t.Fatalf("read first establishment social links: %v", err)
	}
	if len(links) != 2 || links[0].Position != 0 || links[0].Label != "Site" || links[1].Position != 1 || links[1].Label != "Instagram" {
		t.Fatalf("social links are not returned in position order: %+v", links)
	}
	clearTransaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin social link replacement transaction: %v", err)
	}
	t.Cleanup(func() { _ = clearTransaction.Rollback(ctx) })
	if err := db.New(clearTransaction).DeleteEstablishmentSocialLinks(ctx, pgtype.UUID{Bytes: firstTenant, Valid: true}); err != nil {
		t.Fatalf("clear first establishment social links: %v", err)
	}
	if err := clearTransaction.Commit(ctx); err != nil {
		t.Fatalf("commit social link replacement: %v", err)
	}
	links, err = queries.ListEstablishmentSocialLinks(ctx, pgtype.UUID{Bytes: firstTenant, Valid: true})
	if err != nil {
		t.Fatalf("read cleared establishment social links: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("cleared social links remain: %+v", links)
	}

	otherSettings, err := queries.GetEstablishmentPublicSettings(ctx, pgtype.UUID{Bytes: secondTenant, Valid: true})
	if err != nil {
		t.Fatalf("read second establishment settings: %v", err)
	}
	if otherSettings.Description.Valid || otherSettings.Address.Valid || otherSettings.ContactPhone.Valid ||
		otherSettings.MinimumBookingNoticeMinutes.Valid || otherSettings.BookingHorizonDays.Valid || otherSettings.CustomerChangeDeadlineMinutes.Valid {
		t.Fatalf("first tenant settings leaked into second tenant: %+v", otherSettings)
	}
	otherLinks, err := queries.ListEstablishmentSocialLinks(ctx, pgtype.UUID{Bytes: secondTenant, Valid: true})
	if err != nil {
		t.Fatalf("read second establishment social links: %v", err)
	}
	if len(otherLinks) != 0 {
		t.Fatalf("second tenant received another tenant's social links: %+v", otherLinks)
	}
}
