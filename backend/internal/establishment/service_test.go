package establishment

import (
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
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

func TestServiceReturnsSettingsOnlyForOwnerAndDerivesAccessFlags(t *testing.T) {
	store := &settingsStoreStub{settings: EstablishmentSettings{
		Establishment: Establishment{
			ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
		}, Description: stringPointer("Descrição"), OperationalDataStarted: false,
	}}
	service := NewSettingsService(store, writeEntitlementsStub{})

	result, err := service.GetSettings(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID, Role: "owner"})

	if err != nil {
		t.Fatal(err)
	}
	if store.establishmentID != testEstablishmentID {
		t.Fatalf("queried establishment = %q", store.establishmentID)
	}
	if !result.TimeZoneEditable || !result.CanWrite || result.SlugConflict || result.IsPublishable || result.IsComplete {
		t.Fatalf("unexpected settings flags: %+v", result)
	}
}

func TestServiceRejectsProfessionalSettingsAccess(t *testing.T) {
	store := &settingsStoreStub{}
	service := NewSettingsService(store, writeEntitlementsStub{})

	_, err := service.GetSettings(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID, Role: "professional"})

	if !errors.Is(err, ErrSettingsForbidden) {
		t.Fatalf("error = %v, want ErrSettingsForbidden", err)
	}
	if store.establishmentID != "" {
		t.Fatalf("store queried for unauthorized role: %q", store.establishmentID)
	}
}

func TestServiceSettingsPreserveOptionalFieldsAndMarkPublicationState(t *testing.T) {
	store := &settingsStoreStub{settings: EstablishmentSettings{
		Establishment: Establishment{ID: testEstablishmentID, Name: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo"},
		SocialLinks:   []SocialLink{{Label: "Instagram", URL: "https://instagram.com/example"}},
		Logo:          &LogoMetadata{ContentType: "image/png", ByteSize: 120, Width: 20, Height: 10},
	}}
	service := NewSettingsService(store, writeEntitlementsStub{})

	result, err := service.GetSettings(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID, Role: "owner"})

	if err != nil {
		t.Fatal(err)
	}
	if !result.IsComplete || !result.IsPublishable || result.SlugConflict || result.Logo == nil || len(result.SocialLinks) != 1 {
		t.Fatalf("unexpected publication settings: %+v", result)
	}
}

func TestServiceSettingsReportLockedTimezoneAndReadOnlyAccess(t *testing.T) {
	store := &settingsStoreStub{settings: EstablishmentSettings{
		Establishment:          Establishment{ID: testEstablishmentID, Name: "Legado", Slug: "admin", TimeZone: "America/Sao_Paulo"},
		OperationalDataStarted: true,
		Logo:                   &LogoMetadata{ContentType: "image/png", ByteSize: 120, Width: 20, Height: 10},
	}}
	service := NewSettingsService(store, writeEntitlementsStub{err: entitlements.ErrReadOnly})

	result, err := service.GetSettings(context.Background(), auth.Principal{EstablishmentID: testEstablishmentID, Role: "owner"})

	if err != nil {
		t.Fatal(err)
	}
	if result.CanWrite || result.TimeZoneEditable || !result.SlugConflict || result.IsPublishable || result.Logo == nil || result.Logo.URL != nil {
		t.Fatalf("unexpected restricted settings flags: %+v", result)
	}
}

func stringPointer(value string) *string { return &value }

type storeStub struct {
	establishmentID string
	establishment   Establishment
	err             error
}

type settingsStoreStub struct {
	establishmentID string
	settings        EstablishmentSettings
	err             error
}

type writeEntitlementsStub struct {
	err error
}

func (stub writeEntitlementsStub) RequireWrite(context.Context, string) error { return stub.err }

func (stub *settingsStoreStub) GetSettingsByID(_ context.Context, establishmentID string) (EstablishmentSettings, error) {
	stub.establishmentID = establishmentID
	return stub.settings, stub.err
}

func (stub *storeStub) GetByID(_ context.Context, establishmentID string) (Establishment, error) {
	stub.establishmentID = establishmentID
	return stub.establishment, stub.err
}
