package professionals

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"context"
	"errors"
	"testing"
	"time"
)

type deactivationStoreStub struct {
	createStoreStub
	tenant, id string
}

func (stub *deactivationStoreStub) Deactivate(_ context.Context, tenant, id string, _ time.Time) (Professional, error) {
	stub.tenant, stub.id = tenant, id
	return Professional{ID: id, Active: false}, nil
}
func TestDeactivateRequiresOwnerAndWriteEntitlement(t *testing.T) {
	for _, test := range []struct {
		name, role     string
		writeErr, want error
	}{
		{name: "owner", role: "owner"},
		{name: "professional", role: "professional", want: ErrForbidden},
		{name: "read only", role: "owner", writeErr: entitlements.ErrReadOnly, want: entitlements.ErrReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &deactivationStoreStub{}
			service := NewService(store, &passwordHasherStub{}, &writeEntitlementsStub{err: test.writeErr})
			record, err := service.Deactivate(context.Background(), auth.Principal{Role: test.role, EstablishmentID: establishmentID}, "professional-id")
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v", err)
			}
			if err == nil && (store.tenant != establishmentID || store.id != "professional-id" || record.Active) {
				t.Fatal("deactivation used incorrect identity or response")
			}
			if err != nil && store.id != "" {
				t.Fatal("refused operation reached persistence")
			}
		})
	}
}
