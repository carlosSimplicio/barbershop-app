package establishment

import (
	"context"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
)

const (
	testEstablishmentID  = "018f47d2-e886-7f1c-8e8d-8b38eaf16464"
	otherEstablishmentID = "018f47d2-e886-7f1c-8e8d-8b38eaf16465"
)

func TestServiceReturnsAuthenticatedEstablishmentAndActiveTrialInUTC(t *testing.T) {
	startsAt := time.Date(2026, time.October, 19, 11, 30, 0, 0, time.FixedZone("BRT", -3*60*60))
	endsAt := startsAt.Add(14 * 24 * time.Hour)
	store := &storeStub{establishment: Establishment{
		ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
		Trial: &Trial{StartsAt: startsAt, EndsAt: endsAt},
	}}
	service := NewService(store)
	service.now = func() time.Time { return startsAt.Add(time.Hour) }

	result, err := service.Get(context.Background(), auth.Principal{
		UserID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: testEstablishmentID, Role: "owner",
	})

	if err != nil {
		t.Fatal(err)
	}
	if store.establishmentID != testEstablishmentID {
		t.Fatalf("queried establishment = %q", store.establishmentID)
	}
	if result.Trial == nil || result.Trial.Status != TrialStatusActive {
		t.Fatalf("trial = %#v", result.Trial)
	}
	if result.Trial.StartsAt.Location() != time.UTC || result.Trial.EndsAt.Location() != time.UTC {
		t.Fatalf("trial limits are not UTC: %#v", result.Trial)
	}
	if !result.Trial.StartsAt.Equal(startsAt) || !result.Trial.EndsAt.Equal(endsAt) {
		t.Fatalf("trial limits changed: %#v", result.Trial)
	}
}

func TestServiceReturnsNoTrialBeforeConfirmation(t *testing.T) {
	service := NewService(&storeStub{establishment: Establishment{
		ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
	}})

	result, err := service.Get(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID})

	if err != nil {
		t.Fatal(err)
	}
	if result.Trial != nil {
		t.Fatalf("trial = %#v, want nil", result.Trial)
	}
}

func TestServiceExpiresTrialAtExactEnd(t *testing.T) {
	endsAt := time.Date(2026, time.November, 2, 14, 30, 0, 0, time.UTC)
	service := NewService(&storeStub{establishment: Establishment{
		ID:    testEstablishmentID,
		Trial: &Trial{StartsAt: endsAt.Add(-14 * 24 * time.Hour), EndsAt: endsAt},
	}})
	service.now = func() time.Time { return endsAt }

	result, err := service.Get(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID})

	if err != nil {
		t.Fatal(err)
	}
	if result.Trial == nil || result.Trial.Status != TrialStatusExpired {
		t.Fatalf("trial = %#v", result.Trial)
	}
}

type storeStub struct {
	establishmentID string
	establishment   Establishment
	err             error
}

func (stub *storeStub) GetByID(_ context.Context, establishmentID string) (Establishment, error) {
	stub.establishmentID = establishmentID
	return stub.establishment, stub.err
}
