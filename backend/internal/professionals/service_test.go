package professionals

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
)

const establishmentID = "018f47d2-e886-7f1c-8e8d-8b38eaf16464"

func TestCreateProfessionalValidatesAndPersistsOwnerInput(t *testing.T) {
	createdAt := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	store := &createStoreStub{id: "018f47d2-e886-7f1c-8e8d-8b38eaf16466"}
	passwords := &passwordHasherStub{}
	writes := &writeEntitlementsStub{}
	service := NewService(store, passwords, writes)
	service.now = func() time.Time { return createdAt }

	result, err := service.Create(context.Background(), auth.Principal{Role: "owner", EstablishmentID: establishmentID}, CreateInput{
		Name: "  Ana Silva  ", Email: "  ANA@Example.test  ", TemporaryPassword: "senha-segura",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != store.id || result.Name != "Ana Silva" || result.Email != "ANA@Example.test" || !result.Active {
		t.Fatalf("result = %#v", result)
	}
	if store.creation.EstablishmentID != establishmentID || store.creation.Name != result.Name || store.creation.Email != result.Email ||
		store.creation.NormalizedEmail != "ana@example.test" || store.creation.PasswordHash != "argon2id-hash" || !store.creation.CreatedAt.Equal(createdAt) {
		t.Fatalf("persisted creation = %#v", store.creation)
	}
	if writes.establishmentID != establishmentID || passwords.password != "senha-segura" {
		t.Fatal("trial or password hash used unexpected input")
	}
}

func TestCreateProfessionalRejectsInvalidFieldsBeforeWrite(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
		want  error
	}{
		{name: "blank name", input: CreateInput{Name: "  ", Email: "ana@example.test", TemporaryPassword: "senha-segura"}, want: ErrInvalidName},
		{name: "long name", input: CreateInput{Name: strings.Repeat("a", 121), Email: "ana@example.test", TemporaryPassword: "senha-segura"}, want: ErrInvalidName},
		{name: "invalid email", input: CreateInput{Name: "Ana", Email: "invalid", TemporaryPassword: "senha-segura"}, want: auth.ErrInvalidEmail},
		{name: "invalid password", input: CreateInput{Name: "Ana", Email: "ana@example.test", TemporaryPassword: "short"}, want: security.ErrInvalidPassword},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &createStoreStub{}
			service := NewService(store, &passwordHasherStub{}, &writeEntitlementsStub{})
			_, err := service.Create(context.Background(), auth.Principal{Role: "owner", EstablishmentID: establishmentID}, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if store.called {
				t.Fatal("invalid input was persisted")
			}
		})
	}
}

func TestCreateProfessionalRejectsNonOwnerAndReadOnlyEstablishment(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		writeErr error
		want     error
	}{
		{name: "professional", role: "professional", want: ErrForbidden},
		{name: "expired trial", role: "owner", writeErr: entitlements.ErrReadOnly, want: entitlements.ErrReadOnly},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &createStoreStub{}
			writes := &writeEntitlementsStub{err: test.writeErr}
			service := NewService(store, &passwordHasherStub{}, writes)
			_, err := service.Create(context.Background(), auth.Principal{Role: test.role, EstablishmentID: establishmentID}, CreateInput{
				Name: "Ana", Email: "ana@example.test", TemporaryPassword: "senha-segura",
			})
			if !errors.Is(err, test.want) || store.called {
				t.Fatalf("error = %v, persisted = %v", err, store.called)
			}
		})
	}
}

type createStoreStub struct {
	id       string
	creation Creation
	called   bool
}

func (stub *createStoreStub) Create(_ context.Context, creation Creation) (string, error) {
	stub.called = true
	stub.creation = creation
	return stub.id, nil
}

func (stub *createStoreStub) List(context.Context, string, int32, int32) ([]Professional, error) {
	return nil, nil
}

type passwordHasherStub struct{ password string }

func (stub *passwordHasherStub) Hash(password string) (string, error) {
	stub.password = password
	return "argon2id-hash", nil
}

type writeEntitlementsStub struct {
	establishmentID string
	err             error
}

func (stub *writeEntitlementsStub) RequireWrite(_ context.Context, establishmentID string) error {
	stub.establishmentID = establishmentID
	return stub.err
}
