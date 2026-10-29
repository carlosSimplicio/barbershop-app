package professionals

import (
	"context"
	"errors"
	"testing"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
)

func TestListProfessionalsLimitsPageAndDerivesTenantFromOwner(t *testing.T) {
	store := &listStoreStub{items: []Professional{
		{ID: "first", Name: "Ana"}, {ID: "second", Name: "Bia"}, {ID: "third", Name: "Caio"},
	}}
	writes := &writeEntitlementsStub{err: entitlements.ErrReadOnly}
	service := NewService(store, &passwordHasherStub{}, writes)

	result, err := service.List(context.Background(), auth.Principal{Role: "owner", EstablishmentID: establishmentID}, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 2 || result.Offset != 3 || !result.HasMore || len(result.Items) != 2 ||
		result.Items[0].ID != "first" || result.Items[1].ID != "second" {
		t.Fatalf("list result = %#v", result)
	}
	if store.establishmentID != establishmentID || store.limit != 3 || store.offset != 3 {
		t.Fatalf("query tenant=%q limit=%d offset=%d", store.establishmentID, store.limit, store.offset)
	}
	if writes.establishmentID != establishmentID {
		t.Fatal("read did not resolve write entitlement")
	}
}

func TestListProfessionalsReturnsEmptyPage(t *testing.T) {
	service := NewService(&listStoreStub{}, &passwordHasherStub{}, &writeEntitlementsStub{})
	result, err := service.List(context.Background(), auth.Principal{Role: "owner", EstablishmentID: establishmentID}, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil || len(result.Items) != 0 || result.HasMore {
		t.Fatalf("empty list = %#v", result)
	}
}

func TestListProfessionalsRejectsRoleAndInvalidPagination(t *testing.T) {
	for _, test := range []struct {
		name   string
		role   string
		limit  int32
		offset int32
		want   error
	}{
		{name: "professional", role: "professional", limit: 20, want: ErrForbidden},
		{name: "zero limit", role: "owner", want: ErrInvalidLimit},
		{name: "over limit", role: "owner", limit: 101, want: ErrInvalidLimit},
		{name: "negative offset", role: "owner", limit: 20, offset: -1, want: ErrInvalidOffset},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &listStoreStub{}
			service := NewService(store, &passwordHasherStub{}, &writeEntitlementsStub{})
			_, err := service.List(context.Background(), auth.Principal{Role: test.role, EstablishmentID: establishmentID}, test.limit, test.offset)
			if !errors.Is(err, test.want) || store.called {
				t.Fatalf("error = %v, queried = %v", err, store.called)
			}
		})
	}
}

type listStoreStub struct {
	items           []Professional
	establishmentID string
	limit           int32
	offset          int32
	called          bool
}

func (*listStoreStub) Create(context.Context, Creation) (string, error) { return "", nil }

func (stub *listStoreStub) List(_ context.Context, establishmentID string, limit, offset int32) ([]Professional, error) {
	stub.called = true
	stub.establishmentID = establishmentID
	stub.limit = limit
	stub.offset = offset
	return stub.items, nil
}

func TestListProfessionalsDistinguishesReadOnlyFromInfrastructureFailure(t *testing.T) {
	infrastructureErr := errors.New("entitlements unavailable")
	for _, test := range []struct {
		name     string
		writeErr error
		wantErr  error
	}{
		{name: "active"}, {name: "read only", writeErr: entitlements.ErrReadOnly},
		{name: "infrastructure", writeErr: infrastructureErr, wantErr: infrastructureErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(&listStoreStub{}, &passwordHasherStub{}, &writeEntitlementsStub{err: test.writeErr})
			result, err := service.List(context.Background(), auth.Principal{Role: "owner", EstablishmentID: establishmentID}, 20, 0)
			if err == nil && result.CanWrite != (test.writeErr == nil) {
				t.Fatalf("can write = %v", result.CanWrite)
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
