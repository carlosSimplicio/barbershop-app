package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"barbershop-app/backend/internal/auth/security"
)

type recoveryVerificationStoreStub struct {
	called   bool
	hash     []byte
	password string
	err      error
}

func (stub *recoveryVerificationStoreStub) ResetOwnerPassword(_ context.Context, hash []byte, password string, _ PasswordManager) error {
	stub.called, stub.hash, stub.password = true, hash, password
	return stub.err
}

func TestPasswordRecoveryVerificationRejectsMalformedTokenBeforeStore(t *testing.T) {
	store := &recoveryVerificationStoreStub{}
	service := NewPasswordRecoveryVerificationService(store, nil)
	err := service.Verify(context.Background(), "invalid-token", "nova-senha-segura")
	if !errors.Is(err, ErrInvalidPasswordRecovery) || store.called {
		t.Fatalf("error = %v, store called = %t", err, store.called)
	}
}

func TestPasswordRecoveryVerificationRejectsInvalidPasswordBeforeStore(t *testing.T) {
	store := &recoveryVerificationStoreStub{}
	service := NewPasswordRecoveryVerificationService(store, nil)
	token := testOpaqueRecoveryToken(t)
	err := service.Verify(context.Background(), token, "curta")
	if !errors.Is(err, security.ErrInvalidPassword) || store.called {
		t.Fatalf("error = %v, store called = %t", err, store.called)
	}
}

func TestPasswordRecoveryVerificationHashesValidTokenForStore(t *testing.T) {
	store := &recoveryVerificationStoreStub{}
	service := NewPasswordRecoveryVerificationService(store, nil)
	token := testOpaqueRecoveryToken(t)
	err := service.Verify(context.Background(), token, "nova-senha-segura")
	if err != nil || !store.called || store.password != "nova-senha-segura" || !bytes.Equal(store.hash, security.HashOpaqueToken(token)) || bytes.Equal(store.hash, []byte(token)) {
		t.Fatalf("error = %v, store = %+v", err, store)
	}
}

func testOpaqueRecoveryToken(t *testing.T) string {
	t.Helper()
	token, err := security.NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
