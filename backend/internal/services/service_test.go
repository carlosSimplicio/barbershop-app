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

func (s *testStore) List(_ context.Context, tenant string, limit, offset int32) ([]Record, error) {
	s.tenant = tenant
	s.called = true
	return []Record{{ID: "first"}, {ID: "second"}, {ID: "third"}}, s.err
}
func TestServiceListReturnsBoundedPageAndWriteRights(t *testing.T) {
	for _, tc := range []struct {
		err      error
		canWrite bool
	}{{nil, true}, {entitlements.ErrReadOnly, false}} {
		store := &testStore{}
		page, err := NewService(store, testWrites{tc.err}).List(context.Background(), auth.Principal{Role: "owner", EstablishmentID: "tenant"}, 2, 0)
		if err != nil || len(page.Items) != 2 || !page.HasMore || page.CanWrite != tc.canWrite || store.tenant != "tenant" {
			t.Fatalf("page=%+v err=%v", page, err)
		}
	}
	failure := errors.New("storage failure")
	if _, err := NewService(&testStore{}, testWrites{failure}).List(context.Background(), auth.Principal{Role: "owner"}, 20, 0); !errors.Is(err, failure) {
		t.Fatalf("entitlement failure=%v", err)
	}
	for _, tc := range []struct {
		role          string
		limit, offset int64
		want          error
	}{{"professional", 20, 0, ErrForbidden}, {"owner", 0, 0, ErrInvalidLimit}, {"owner", 101, 0, ErrInvalidLimit}, {"owner", 20, -1, ErrInvalidOffset}, {"owner", 20, 2147483648, ErrInvalidOffset}} {
		store := &testStore{}
		_, err := NewService(store, testWrites{}).List(context.Background(), auth.Principal{Role: tc.role}, tc.limit, tc.offset)
		if !errors.Is(err, tc.want) || store.called {
			t.Fatalf("error=%v called=%v", err, store.called)
		}
	}
}

func (s *testStore) Get(_ context.Context, tenant, id string) (Record, error) {
	s.called = true
	s.tenant = tenant
	return Record{ID: id}, s.err
}
func TestServiceGetAuthorizesAndPropagatesMissingResource(t *testing.T) {
	for _, tc := range []struct {
		role           string
		storeErr, want error
	}{{"owner", nil, nil}, {"owner", ErrNotFound, ErrNotFound}, {"professional", nil, ErrForbidden}} {
		store := &testStore{err: tc.storeErr}
		record, err := NewService(store, testWrites{entitlements.ErrReadOnly}).Get(context.Background(), auth.Principal{Role: tc.role, EstablishmentID: "tenant"}, "service")
		if !errors.Is(err, tc.want) {
			t.Fatalf("err=%v want=%v", err, tc.want)
		}
		if tc.want == nil && (record.ID != "service" || store.tenant != "tenant") {
			t.Fatalf("record=%+v store=%+v", record, store)
		}
		if tc.role != "owner" && store.called {
			t.Fatal("professional read store")
		}
	}
}
