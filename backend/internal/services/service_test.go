package services

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"context"
	"errors"
	"testing"
)

type testStore struct {
	tenant, name    string
	price, duration int32
	called          bool
	err             error
}

func (s *testStore) Create(_ context.Context, tenant, name string, price, duration int32) (Record, error) {
	s.called = true
	s.tenant = tenant
	s.name = name
	s.price = price
	s.duration = duration
	return Record{ID: "c2bb0aaf-550b-497d-b18e-e16e2404d80c", Name: name, PriceCents: int64(price), DurationMinutes: int64(duration), Active: true}, s.err
}

type testWrites struct{ err error }

func (w testWrites) RequireWrite(context.Context, string) error { return w.err }
func ptr(v int64) *int64                                        { return &v }
func TestServiceCreateNormalizesAndUsesSessionTenant(t *testing.T) {
	store := &testStore{}
	s := NewService(store, testWrites{})
	record, err := s.Create(context.Background(), auth.Principal{Role: "owner", EstablishmentID: "tenant"}, CreateInput{Name: " Corte ", PriceCents: ptr(1)})
	if err != nil || record.Name != "Corte" || record.DurationMinutes != 30 || !record.Active || store.tenant != "tenant" || store.price != 1 {
		t.Fatalf("record=%+v err=%v store=%+v", record, err, store)
	}
}
func TestServiceCreateRefusesInvalidOrUnauthorizedWrites(t *testing.T) {
	for _, tc := range []struct {
		role           string
		in             CreateInput
		writeErr, want error
	}{
		{"professional", CreateInput{Name: "Corte", PriceCents: ptr(0)}, nil, ErrForbidden},
		{"owner", CreateInput{Name: "Corte", PriceCents: ptr(0)}, entitlements.ErrReadOnly, entitlements.ErrReadOnly},
		{"owner", CreateInput{Name: " ", PriceCents: ptr(0)}, nil, ErrInvalidName},
		{"owner", CreateInput{Name: "Corte"}, nil, ErrInvalidPrice},
		{"owner", CreateInput{Name: "Corte", PriceCents: ptr(-1)}, nil, ErrInvalidPrice},
		{"owner", CreateInput{Name: "Corte", PriceCents: ptr(0), DurationMinutes: ptr(0)}, nil, ErrInvalidDuration},
	} {
		store := &testStore{}
		_, err := NewService(store, testWrites{tc.writeErr}).Create(context.Background(), auth.Principal{Role: tc.role}, tc.in)
		if !errors.Is(err, tc.want) || store.called {
			t.Errorf("err=%v want=%v persisted=%v", err, tc.want, store.called)
		}
	}
}
