package entitlements

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testEstablishmentID = "018f47d2-e886-7f1c-8e8d-8b38eaf16464"

func TestRequireWriteUsesActiveTrialWindow(t *testing.T) {
	startsAt := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(14 * 24 * time.Hour)

	tests := []struct {
		name    string
		trial   *TrialWindow
		now     time.Time
		allowed bool
	}{
		{name: "no trial", now: startsAt},
		{name: "before start", trial: &TrialWindow{StartsAt: startsAt, EndsAt: endsAt}, now: startsAt.Add(-time.Nanosecond)},
		{name: "at start", trial: &TrialWindow{StartsAt: startsAt, EndsAt: endsAt}, now: startsAt, allowed: true},
		{name: "before end", trial: &TrialWindow{StartsAt: startsAt, EndsAt: endsAt}, now: endsAt.Add(-time.Nanosecond), allowed: true},
		{name: "at end", trial: &TrialWindow{StartsAt: startsAt, EndsAt: endsAt}, now: endsAt},
		{name: "after end", trial: &TrialWindow{StartsAt: startsAt, EndsAt: endsAt}, now: endsAt.Add(time.Nanosecond)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &trialStoreStub{trial: test.trial}
			service := NewService(store)
			service.now = func() time.Time { return test.now }

			err := service.RequireWrite(context.Background(), testEstablishmentID)
			if test.allowed && err != nil {
				t.Fatalf("active trial refused write: %v", err)
			}
			if !test.allowed && !errors.Is(err, ErrReadOnly) {
				t.Fatalf("inactive trial error = %v, want ErrReadOnly", err)
			}
			if store.establishmentID != testEstablishmentID {
				t.Fatalf("queried establishment = %q", store.establishmentID)
			}
		})
	}
}

func TestRequireWriteFailsClosedWhenTrialLookupFails(t *testing.T) {
	lookupErr := errors.New("database unavailable")
	service := NewService(&trialStoreStub{err: lookupErr})

	err := service.RequireWrite(context.Background(), testEstablishmentID)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("lookup error = %v, want database failure", err)
	}
}

type trialStoreStub struct {
	establishmentID string
	trial           *TrialWindow
	err             error
}

func (stub *trialStoreStub) GetTrialWindow(_ context.Context, establishmentID string) (*TrialWindow, error) {
	stub.establishmentID = establishmentID
	return stub.trial, stub.err
}
