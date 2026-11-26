package offerings

import (
	"context"
	"errors"
	"testing"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/catalog"
	"barbershop-app/backend/internal/entitlements"
)

type storeStub struct {
	professional           Professional
	professionalErr        error
	items                  []Offering
	listErr                error
	tenant, professionalID string
	limit, offset          int32
	queried                bool
}

func (stub *storeStub) GetProfessional(_ context.Context, tenant, professionalID string) (Professional, error) {
	stub.queried = true
	stub.tenant = tenant
	stub.professionalID = professionalID
	return stub.professional, stub.professionalErr
}
func (stub *storeStub) List(_ context.Context, tenant, professionalID string, limit, offset int32) ([]Offering, error) {
	stub.tenant = tenant
	stub.professionalID = professionalID
	stub.limit = limit
	stub.offset = offset
	return stub.items, stub.listErr
}

type writesStub struct {
	err    error
	tenant string
}

func (stub *writesStub) RequireWrite(_ context.Context, tenant string) error {
	stub.tenant = tenant
	return stub.err
}

func TestListReturnsBoundedCatalogAndProfessionalWithoutRequiringWrite(t *testing.T) {
	for _, writeErr := range []error{nil, entitlements.ErrReadOnly} {
		store := &storeStub{professional: Professional{ID: "professional", Name: "Ana", Active: false}, items: []Offering{{Offering: catalog.Offering{ID: "first"}, Associated: true}, {Offering: catalog.Offering{ID: "second"}}, {Offering: catalog.Offering{ID: "extra"}}}}
		writes := &writesStub{err: writeErr}
		result, err := NewService(store, writes).List(context.Background(), auth.Principal{Role: "owner", EstablishmentID: "tenant"}, "professional", 2, 3)
		if err != nil || result.Professional.ID != "professional" || result.Professional.Active || len(result.Items) != 2 || !result.Items[0].Associated || !result.HasMore || result.Limit != 2 || result.Offset != 3 || result.CanWrite != (writeErr == nil) {
			t.Fatalf("page=%#v err=%v", result, err)
		}
		if store.tenant != "tenant" || store.professionalID != "professional" || store.limit != 3 || store.offset != 3 || writes.tenant != "tenant" {
			t.Fatalf("wrong tenant/page: %#v %#v", store, writes)
		}
	}
}

func TestListRejectsRoleAndPaginationBeforeReading(t *testing.T) {
	for _, test := range []struct {
		role          string
		limit, offset int64
		want          error
	}{
		{"professional", 20, 0, ErrForbidden}, {"", 20, 0, ErrForbidden}, {"owner", 0, 0, ErrInvalidLimit}, {"owner", 101, 0, ErrInvalidLimit}, {"owner", 20, -1, ErrInvalidOffset}, {"owner", 20, 2147483648, ErrInvalidOffset},
	} {
		store := &storeStub{}
		_, err := NewService(store, &writesStub{}).List(context.Background(), auth.Principal{Role: test.role}, "professional", test.limit, test.offset)
		if !errors.Is(err, test.want) || store.queried {
			t.Fatalf("error=%v queried=%v want=%v", err, store.queried, test.want)
		}
	}
}

func TestListPropagatesMissingProfessionalAndInfrastructureErrors(t *testing.T) {
	unavailable := errors.New("unavailable")
	for _, test := range []struct{ professionalErr, writeErr, listErr, want error }{
		{ErrNotFound, nil, nil, ErrNotFound}, {unavailable, nil, nil, unavailable}, {nil, unavailable, nil, unavailable}, {nil, nil, unavailable, unavailable},
	} {
		_, err := NewService(&storeStub{professionalErr: test.professionalErr, listErr: test.listErr}, &writesStub{err: test.writeErr}).List(context.Background(), auth.Principal{Role: "owner"}, "professional", 20, 0)
		if !errors.Is(err, test.want) {
			t.Fatalf("error=%v want=%v", err, test.want)
		}
	}
}

func TestListEmptyCatalogIsArrayAndAcceptsPaginationBounds(t *testing.T) {
	result, err := NewService(&storeStub{}, &writesStub{}).List(context.Background(), auth.Principal{Role: "owner"}, "professional", 100, 2147483647)
	if err != nil || result.Items == nil || result.HasMore || len(result.Items) != 0 {
		t.Fatalf("page=%#v err=%v", result, err)
	}
}

func (stub *storeStub) Set(_ context.Context, tenant, professionalID, offeringID string, associated bool) (Offering, error) {
	stub.queried = true
	stub.tenant = tenant
	stub.professionalID = professionalID
	return Offering{Offering: catalog.Offering{ID: offeringID}, Associated: associated}, stub.listErr
}
func TestSetRequiresOwnerInputAndWriteBeforePersistence(t *testing.T) {
	unavailable := errors.New("unavailable")
	yes := true
	for _, test := range []struct {
		role           string
		associated     *bool
		writeErr, want error
	}{
		{"professional", &yes, nil, ErrForbidden}, {"owner", nil, nil, ErrInvalidRequest}, {"owner", &yes, entitlements.ErrReadOnly, entitlements.ErrReadOnly}, {"owner", &yes, unavailable, unavailable},
	} {
		store := &storeStub{}
		_, err := NewService(store, &writesStub{err: test.writeErr}).Set(context.Background(), auth.Principal{Role: test.role}, "professional", "offering", test.associated)
		if !errors.Is(err, test.want) || store.queried {
			t.Fatalf("error=%v persisted=%v want=%v", err, store.queried, test.want)
		}
	}
}
func TestSetPassesExplicitStateAndTenantAndPropagatesMissingTarget(t *testing.T) {
	for _, associated := range []bool{true, false} {
		store := &storeStub{}
		writes := &writesStub{}
		record, err := NewService(store, writes).Set(context.Background(), auth.Principal{Role: "owner", EstablishmentID: "tenant"}, "professional", "offering", &associated)
		if err != nil || record.ID != "offering" || record.Associated != associated || store.tenant != "tenant" || store.professionalID != "professional" || writes.tenant != "tenant" {
			t.Fatalf("record=%+v err=%v store=%+v", record, err, store)
		}
	}
	yes := true
	_, err := NewService(&storeStub{listErr: ErrNotFound}, &writesStub{}).Set(context.Background(), auth.Principal{Role: "owner"}, "professional", "offering", &yes)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing target=%v", err)
	}
}
