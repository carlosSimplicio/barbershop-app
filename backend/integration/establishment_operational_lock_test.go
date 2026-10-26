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
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestOperationalMarkerMigrationPreservesSettingsAndProtectsRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaName := "operational_lock_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	if err := goose.UpToContext(ctx, database, migrationDirectory, 6); err != nil {
		t.Fatalf("apply baseline migrations: %v", err)
	}
	var establishmentID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO establishments (name, slug, time_zone, description, address, contact_phone, minimum_booking_notice_minutes)
		VALUES ('Barbearia Inicial', 'barbearia-inicial', 'America/Sao_Paulo', 'Cortes', 'Rua Um, 10', '+5511999990000', 30)
		RETURNING id
	`).Scan(&establishmentID); err != nil {
		t.Fatalf("create establishment before marker migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO establishment_social_links (establishment_id, position, label, url)
		VALUES ($1, 0, 'Instagram', 'https://instagram.com/barbearia-inicial')
	`, establishmentID); err != nil {
		t.Fatalf("create social link before marker migration: %v", err)
	}
	if err := goose.UpContext(ctx, database, migrationDirectory); err != nil {
		t.Fatalf("apply operational marker migration: %v", err)
	}
	var name, slug, timeZone, description, address, phone sql.NullString
	var minimumNotice sql.NullInt64
	var marker sql.NullTime
	if err := pool.QueryRow(ctx, `
		SELECT name, slug, time_zone, description, address, contact_phone,
			minimum_booking_notice_minutes, operational_data_started_at
		FROM establishments WHERE id = $1
	`, establishmentID).Scan(&name, &slug, &timeZone, &description, &address, &phone, &minimumNotice, &marker); err != nil {
		t.Fatalf("read establishment after marker migration: %v", err)
	}
	if !name.Valid || name.String != "Barbearia Inicial" || !slug.Valid || slug.String != "barbearia-inicial" ||
		!timeZone.Valid || timeZone.String != "America/Sao_Paulo" || !description.Valid || description.String != "Cortes" ||
		!address.Valid || address.String != "Rua Um, 10" || !phone.Valid || phone.String != "+5511999990000" ||
		!minimumNotice.Valid || minimumNotice.Int64 != 30 || marker.Valid {
		t.Fatalf("marker migration changed saved settings or initialized the lock: name=%+v slug=%+v time_zone=%+v description=%+v address=%+v phone=%+v notice=%+v marker=%+v",
			name, slug, timeZone, description, address, phone, minimumNotice, marker)
	}
	var socialLinkCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM establishment_social_links WHERE establishment_id = $1", establishmentID).Scan(&socialLinkCount); err != nil {
		t.Fatalf("read social links after marker migration: %v", err)
	}
	if socialLinkCount != 1 {
		t.Fatalf("social links after marker migration = %d, want 1", socialLinkCount)
	}
	firstTime := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, "UPDATE establishments SET operational_data_started_at = $2 WHERE id = $1", establishmentID, firstTime); err != nil {
		t.Fatalf("set marker before rollback check: %v", err)
	}
	if err := goose.DownToContext(ctx, database, migrationDirectory, 6); err == nil {
		t.Fatal("rollback removed a populated operational marker instead of refusing")
	}
	if err := pool.QueryRow(ctx, "SELECT operational_data_started_at FROM establishments WHERE id = $1", establishmentID).Scan(&marker); err != nil {
		t.Fatalf("read marker after refused rollback: %v", err)
	}
	if !marker.Valid || !marker.Time.Equal(firstTime) {
		t.Fatalf("refused rollback changed marker: %+v", marker)
	}
}

func TestOperationalMarkerIsIdempotentAndRollsBackWithItsTransaction(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	establishmentID := createOperationalMarkerTestEstablishment(t, "operational-marker-transaction")

	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin marker transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback(ctx) })
	transactionQueries := db.New(transaction)
	locked, err := transactionQueries.LockEstablishmentOperationalWrite(ctx, pgtype.UUID{Bytes: establishmentID, Valid: true})
	if err != nil {
		t.Fatalf("lock establishment operational state: %v", err)
	}
	if locked.TimeZone != "America/Sao_Paulo" || locked.OperationalDataStartedAt.Valid {
		t.Fatalf("unexpected unlocked establishment state: %+v", locked)
	}
	firstTime := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Hour)
	markArgs := db.MarkEstablishmentOperationalDataStartedParams{
		ID: establishmentUUID(establishmentID), OperationalDataStartedAt: pgtype.Timestamptz{Time: firstTime, Valid: true},
	}
	if rows, err := transactionQueries.MarkEstablishmentOperationalDataStarted(ctx, markArgs); err != nil || rows != 1 {
		t.Fatalf("mark operational data started: rows=%d err=%v", rows, err)
	}
	markArgs.OperationalDataStartedAt = pgtype.Timestamptz{Time: secondTime, Valid: true}
	if rows, err := transactionQueries.MarkEstablishmentOperationalDataStarted(ctx, markArgs); err != nil || rows != 1 {
		t.Fatalf("mark operational data started again: rows=%d err=%v", rows, err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("commit marker transaction: %v", err)
	}
	var stored sql.NullTime
	if err := integrationPool.QueryRow(ctx, `SELECT operational_data_started_at FROM establishments WHERE id = $1`, establishmentID).Scan(&stored); err != nil {
		t.Fatalf("read committed marker: %v", err)
	}
	if !stored.Valid || !stored.Time.Equal(firstTime) {
		t.Fatalf("committed marker = %+v, want first timestamp %s", stored, firstTime)
	}

	rollbackID := createOperationalMarkerTestEstablishment(t, "operational-marker-rollback")
	rollback, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback marker transaction: %v", err)
	}
	rollbackQueries := db.New(rollback)
	if _, err := rollbackQueries.LockEstablishmentOperationalWrite(ctx, establishmentUUID(rollbackID)); err != nil {
		t.Fatalf("lock establishment before rollback: %v", err)
	}
	if rows, err := rollbackQueries.MarkEstablishmentOperationalDataStarted(ctx, db.MarkEstablishmentOperationalDataStartedParams{
		ID: establishmentUUID(rollbackID), OperationalDataStartedAt: pgtype.Timestamptz{Time: firstTime, Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("mark operational data before rollback: rows=%d err=%v", rows, err)
	}
	if err := rollback.Rollback(ctx); err != nil {
		t.Fatalf("rollback marker transaction: %v", err)
	}
	var rolledBack sql.NullTime
	if err := integrationPool.QueryRow(ctx, `SELECT operational_data_started_at FROM establishments WHERE id = $1`, rollbackID).Scan(&rolledBack); err != nil {
		t.Fatalf("read marker after rollback: %v", err)
	}
	if rolledBack.Valid {
		t.Fatalf("marker survived rolled-back operational transaction: %+v", rolledBack)
	}
}

func TestOperationalMarkerUsesDatabaseTimeWhenCallerHasNoTimestamp(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	establishmentID := createOperationalMarkerTestEstablishment(t, "operational-marker-null-time")
	if rows, err := db.New(integrationPool).MarkEstablishmentOperationalDataStarted(ctx, db.MarkEstablishmentOperationalDataStartedParams{
		ID: establishmentUUID(establishmentID),
	}); err != nil || rows != 1 {
		t.Fatalf("mark operational data without a timestamp: rows=%d err=%v", rows, err)
	}
	var marker sql.NullTime
	if err := integrationPool.QueryRow(ctx, `SELECT operational_data_started_at FROM establishments WHERE id = $1`, establishmentID).Scan(&marker); err != nil {
		t.Fatalf("read marker created without a caller timestamp: %v", err)
	}
	if !marker.Valid {
		t.Fatal("marker remained null after a successful activation")
	}
}

func TestOperationalMarkerLockSerializesFirstOperationalWrite(t *testing.T) {
	resetDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	establishmentID := createOperationalMarkerTestEstablishment(t, "operational-marker-concurrency")
	establishmentIDParam := establishmentUUID(establishmentID)

	firstTransaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	t.Cleanup(func() { _ = firstTransaction.Rollback(context.Background()) })
	firstQueries := db.New(firstTransaction)
	if _, err := firstQueries.LockEstablishmentOperationalWrite(ctx, establishmentIDParam); err != nil {
		t.Fatalf("lock establishment in first transaction: %v", err)
	}

	secondTransaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin second transaction: %v", err)
	}
	t.Cleanup(func() { _ = secondTransaction.Rollback(context.Background()) })
	var secondPID uint32
	if err := secondTransaction.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&secondPID); err != nil {
		t.Fatalf("read second transaction backend PID: %v", err)
	}
	lockResult := make(chan error, 1)
	go func() {
		locked, err := db.New(secondTransaction).LockEstablishmentOperationalWrite(ctx, establishmentIDParam)
		if err == nil && !locked.OperationalDataStartedAt.Valid {
			err = errors.New("operational marker remained unset after waiting for the lock")
		}
		lockResult <- err
	}()
	waitForDatabaseLock(t, ctx, secondPID, lockResult)

	firstTime := time.Date(2026, time.October, 26, 14, 0, 0, 0, time.UTC)
	if rows, err := firstQueries.MarkEstablishmentOperationalDataStarted(ctx, db.MarkEstablishmentOperationalDataStartedParams{
		ID: establishmentIDParam, OperationalDataStartedAt: pgtype.Timestamptz{Time: firstTime, Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("mark operational data in first transaction: rows=%d err=%v", rows, err)
	}
	if err := firstTransaction.Commit(ctx); err != nil {
		t.Fatalf("commit first operational transaction: %v", err)
	}
	select {
	case err := <-lockResult:
		if err != nil {
			t.Fatalf("second transaction did not observe committed marker: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("second transaction remained blocked after first commit: %v", ctx.Err())
	}
}

func createOperationalMarkerTestEstablishment(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := integrationPool.QueryRow(context.Background(), `
		INSERT INTO establishments (name, slug, time_zone)
		VALUES ('Barbearia Trava', $1, 'America/Sao_Paulo')
		RETURNING id
	`, slug).Scan(&id); err != nil {
		t.Fatalf("create establishment %q: %v", slug, err)
	}
	return id
}

func establishmentUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func waitForDatabaseLock(t *testing.T, ctx context.Context, pid uint32, result <-chan error) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waitEvent sql.NullString
		if err := integrationPool.QueryRow(ctx, `
			SELECT wait_event_type
			FROM pg_stat_activity
			WHERE pid = $1
		`, pid).Scan(&waitEvent); err != nil {
			t.Fatalf("observe second transaction lock wait: %v", err)
		}
		if waitEvent.Valid && waitEvent.String == "Lock" {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("second transaction did not wait for the establishment lock: %v", err)
		case <-ctx.Done():
			t.Fatalf("second transaction never waited for the establishment lock: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
