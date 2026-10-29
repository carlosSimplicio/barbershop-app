//go:build integration

package integration_test

import (
	db "barbershop-app/backend/db/generated"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"testing"

	"github.com/google/uuid"
)

func TestCatalogSchemaPreservesExactPriceAndTenant(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	tenant, other := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{tenant, other} {
		if _, err := integrationPool.Exec(ctx, `INSERT INTO establishments (id,name,slug,time_zone) VALUES ($1,'Catálogo fictício',$2,'America/Sao_Paulo')`, id, id.String()); err != nil {
			t.Fatal(err)
		}
	}
	var id uuid.UUID
	if err := integrationPool.QueryRow(ctx, `INSERT INTO services (establishment_id,name,price_cents,duration_minutes) VALUES ($1,'Corte',1,17) RETURNING id`, tenant).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var cents, count int
	if err := integrationPool.QueryRow(ctx, `SELECT price_cents FROM services WHERE id=$1 AND establishment_id=$2`, id, tenant).Scan(&cents); err != nil || cents != 1 {
		t.Fatalf("price=%d err=%v", cents, err)
	}
	if err := integrationPool.QueryRow(ctx, `SELECT count(*) FROM services WHERE id=$1 AND establishment_id=$2`, id, other).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross tenant count=%d err=%v", count, err)
	}
	if _, err := integrationPool.Exec(ctx, `INSERT INTO services (establishment_id,name,price_cents,duration_minutes) VALUES ($1,'Corte',0,30)`, tenant); err != nil {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := integrationPool.Exec(ctx, `INSERT INTO services (establishment_id,name,price_cents,duration_minutes) VALUES ($1,'Corte',0,30)`, uuid.New()); err == nil {
		t.Fatal("orphan service accepted")
	}
	queries := db.New(integrationPool)
	foreign := pgtype.UUID{Bytes: other, Valid: true}
	offeringID := pgtype.UUID{Bytes: id, Valid: true}
	if _, err := queries.GetOffering(ctx, db.GetOfferingParams{ID: offeringID, EstablishmentID: foreign}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("foreign lookup: %v", err)
	}
	rows, err := queries.ListOfferings(ctx, db.ListOfferingsParams{EstablishmentID: foreign, Limit: 20})
	if err != nil || len(rows) != 0 {
		t.Fatalf("foreign list: %v, %v", rows, err)
	}
	if _, err := queries.UpdateOffering(ctx, db.UpdateOfferingParams{ID: offeringID, EstablishmentID: foreign, SetName: true, Name: "Outro"}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("foreign update: %v", err)
	}
	if _, err := queries.SetOfferingStatus(ctx, db.SetOfferingStatusParams{ID: offeringID, EstablishmentID: foreign, Active: false}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("foreign status: %v", err)
	}
	var started bool
	if err := integrationPool.QueryRow(ctx, `SELECT operational_data_started_at IS NOT NULL FROM establishments WHERE id=$1`, tenant).Scan(&started); err != nil || started {
		t.Fatalf("catalog started operational data: %v %v", started, err)
	}
}
