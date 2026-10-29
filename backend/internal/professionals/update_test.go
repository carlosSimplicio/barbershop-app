package professionals

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"context"
	"errors"
	"testing"
	"time"
)

type updateStoreStub struct {
	createStoreStub
	tenant, id, name string
}

func (stub *updateStoreStub) UpdateName(_ context.Context, tenant, id, name string, _ time.Time) (Professional, error) {
	stub.tenant, stub.id, stub.name = tenant, id, name
	return Professional{ID: id, Name: name, Active: false}, nil
}
func TestUpdateProfessionalNameAuthorizesAndNormalizes(t *testing.T) {
	for _, test := range []struct {
		name, role, value string
		writeErr, want    error
	}{
		{name: "valid inactive", role: "owner", value: " Ana "},
		{name: "professional", role: "professional", value: "Ana", want: ErrForbidden},
		{name: "NUL", role: "owner", value: "Ana\x00", want: ErrInvalidName},
		{name: "read only", role: "owner", value: "Ana", writeErr: entitlements.ErrReadOnly, want: entitlements.ErrReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &updateStoreStub{}
			service := NewService(store, &passwordHasherStub{}, &writeEntitlementsStub{err: test.writeErr})
			result, err := service.UpdateName(context.Background(), auth.Principal{Role: test.role, EstablishmentID: establishmentID}, "professional-id", test.value)
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v", err)
			}
			if err == nil && (store.tenant != establishmentID || store.id != "professional-id" || store.name != "Ana" || result.Name != "Ana" || result.Active) {
				t.Fatal("update did not preserve persisted response or normalized tenant input")
			}
			if err != nil && store.id != "" {
				t.Fatal("refused update was persisted")
			}
		})
	}
}
