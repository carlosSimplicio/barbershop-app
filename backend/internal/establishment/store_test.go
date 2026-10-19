package establishment

import (
	"context"
	"testing"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestPostgresStoreGetsEstablishmentByAuthenticatedIDWithTrial(t *testing.T) {
	startsAt := time.Date(2026, time.October, 19, 14, 30, 0, 0, time.UTC)
	endsAt := startsAt.Add(14 * 24 * time.Hour)
	queries := &establishmentQueriesStub{row: db.GetEstablishmentWithTrialRow{
		ID: uuidValue(t, testEstablishmentID), Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
		StartsAt: pgtype.Timestamptz{Time: startsAt, Valid: true}, EndsAt: pgtype.Timestamptz{Time: endsAt, Valid: true},
	}}
	store := &PostgresStore{queries: queries}

	result, err := store.GetByID(context.Background(), testEstablishmentID)

	if err != nil {
		t.Fatal(err)
	}
	if !queries.id.Valid || queries.id.Bytes != uuidValue(t, testEstablishmentID).Bytes {
		t.Fatalf("query id = %#v", queries.id)
	}
	if result.ID != testEstablishmentID || result.Name != "Barbearia Central" || result.Slug != "barbearia-central" || result.TimeZone != "America/Sao_Paulo" ||
		result.Trial == nil || !result.Trial.StartsAt.Equal(startsAt) || !result.Trial.EndsAt.Equal(endsAt) {
		t.Fatalf("establishment = %#v", result)
	}
}

func TestPostgresStoreReturnsNoTrialForNullLimits(t *testing.T) {
	store := &PostgresStore{queries: &establishmentQueriesStub{row: db.GetEstablishmentWithTrialRow{
		ID: uuidValue(t, testEstablishmentID),
	}}}

	result, err := store.GetByID(context.Background(), testEstablishmentID)

	if err != nil || result.Trial != nil {
		t.Fatalf("establishment = %#v, err = %v", result, err)
	}
}

func uuidValue(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	var result pgtype.UUID
	if err := result.Scan(value); err != nil {
		t.Fatal(err)
	}
	return result
}

type establishmentQueriesStub struct {
	id  pgtype.UUID
	row db.GetEstablishmentWithTrialRow
	err error
}

func (stub *establishmentQueriesStub) GetEstablishmentWithTrial(_ context.Context, id pgtype.UUID) (db.GetEstablishmentWithTrialRow, error) {
	stub.id = id
	return stub.row, stub.err
}
