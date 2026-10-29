package professionals

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type reactivationStoreStub struct {
	createStoreStub
	tenant, id, hash string
}

func (stub *reactivationStoreStub) Reactivate(_ context.Context, tenant, id, hash string, _ time.Time) (Professional, error) {
	stub.tenant, stub.id, stub.hash = tenant, id, hash
	return Professional{ID: id, Active: true}, nil
}
func TestReactivateValidatesPasswordAndRequiresOwnerWriteAccess(t *testing.T) {
	for _, test := range []struct {
		name, role, password string
		writeErr, want       error
	}{
		{name: "valid", role: "owner", password: "senha-temporaria"},
		{name: "unicode max", role: "owner", password: strings.Repeat("á", 32)},
		{name: "short", role: "owner", password: "short", want: security.ErrInvalidPassword},
		{name: "long", role: "owner", password: strings.Repeat("á", 33), want: security.ErrInvalidPassword},
		{name: "space", role: "owner", password: "senha temporaria", want: security.ErrInvalidPassword},
		{name: "invisible", role: "owner", password: "senha\u200btemporaria", want: security.ErrInvalidPassword},
		{name: "role", role: "professional", password: "senha-temporaria", want: ErrForbidden},
		{name: "read only", role: "owner", password: "senha-temporaria", writeErr: entitlements.ErrReadOnly, want: entitlements.ErrReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &reactivationStoreStub{}
			hasher := &passwordHasherStub{}
			service := NewService(store, hasher, &writeEntitlementsStub{err: test.writeErr})
			record, err := service.Reactivate(context.Background(), auth.Principal{Role: test.role, EstablishmentID: establishmentID}, "professional-id", test.password)
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v", err)
			}
			if err == nil && (store.tenant != establishmentID || store.id != "professional-id" || store.hash != "argon2id-hash" || hasher.password != test.password || !record.Active) {
				t.Fatal("reactivation did not hash and persist the exact password")
			}
			if err != nil && (store.id != "" || hasher.password != "") {
				t.Fatal("refused reactivation hashed or persisted input")
			}
		})
	}
}
