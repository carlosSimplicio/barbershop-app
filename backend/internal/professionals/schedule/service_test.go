package schedule

import (
	"context"
	"errors"
	"testing"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"

	"github.com/google/uuid"
)

type readStoreStub struct {
	Store
	read func(context.Context, string, uuid.UUID) (Week, error)
}
type writeEntitlementsFunc func(context.Context, string) error

func (stub readStoreStub) Read(ctx context.Context, tenant string, id uuid.UUID) (Week, error) {
	return stub.read(ctx, tenant, id)
}

func (require writeEntitlementsFunc) RequireWrite(ctx context.Context, tenant string) error {
	return require(ctx, tenant)
}

func TestReadScheduleReportsRightsAndKeepsInfrastructureErrors(t *testing.T) {
	tenant, id := uuid.NewString(), uuid.New()
	principal := auth.Principal{Role: "professional", UserID: id.String(), EstablishmentID: tenant}
	store := readStoreStub{read: func(_ context.Context, scope string, target uuid.UUID) (Week, error) {
		if scope != tenant || target != id {
			t.Fatal("read was not scoped to the session and target")
		}
		return Week{TimeZone: "America/Sao_Paulo", Periods: []Period{}}, nil
	}}
	failure := errors.New("entitlements unavailable")
	for _, test := range []struct {
		name                   string
		rightsError, wantError error
		canWrite               bool
	}{
		{name: "active", canWrite: true},
		{name: "expired", rightsError: entitlements.ErrReadOnly},
		{name: "unavailable", rightsError: failure, wantError: failure},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(store, writeEntitlementsFunc(func(_ context.Context, scope string) error {
				if scope != tenant {
					t.Fatal("rights were not scoped to the session")
				}
				return test.rightsError
			}))
			week, err := service.Read(context.Background(), principal, id)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("read error = %v, want %v", err, test.wantError)
			}
			if err == nil && (week.CanWrite != test.canWrite || week.TimeZone != "America/Sao_Paulo" || week.Periods == nil) {
				t.Fatalf("week = %+v", week)
			}
		})
	}
}
