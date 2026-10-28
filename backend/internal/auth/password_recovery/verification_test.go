package passwordrecovery

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
)

type recoveryVerificationStoreStub struct {
	called       bool
	hash         []byte
	user         auth.User
	token        auth.PasswordRecoveryState
	passwordHash string
	err          error
}

func (stub *recoveryVerificationStoreStub) ResetOwnerPassword(_ context.Context, hash []byte, prepare auth.RecoveredPasswordPreparer) error {
	stub.called, stub.hash = true, hash
	if stub.err != nil {
		return stub.err
	}
	passwordHash, err := prepare(stub.user, stub.token)
	if err == nil {
		stub.passwordHash = passwordHash
	}
	return err
}

type recoveryPasswordManagerStub struct {
	unchanged bool
	verifyErr error
	hashErr   error
	verified  bool
	hashed    bool
}

func (stub *recoveryPasswordManagerStub) Verify(_, _ string) (bool, error) {
	stub.verified = true
	return stub.unchanged, stub.verifyErr
}
func (stub *recoveryPasswordManagerStub) Hash(string) (string, error) {
	stub.hashed = true
	return "new-credential-hash", stub.hashErr
}

func validRecoveryVerification() (*PasswordRecoveryVerificationService, *recoveryVerificationStoreStub, *recoveryPasswordManagerStub) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	store := &recoveryVerificationStoreStub{
		user:  auth.User{ID: "owner-id", Role: "owner", Active: true, EmailConfirmed: true, PasswordHash: "previous-credential-hash"},
		token: auth.PasswordRecoveryState{UserID: "owner-id", ExpiresAt: now.Add(time.Hour)},
	}
	passwords := &recoveryPasswordManagerStub{}
	service := NewPasswordRecoveryVerificationService(store, passwords)
	service.now = func() time.Time { return now }
	return service, store, passwords
}

func TestPasswordRecoveryVerificationRejectsMalformedTokenBeforeStore(t *testing.T) {
	service, store, _ := validRecoveryVerification()
	err := service.Verify(context.Background(), "invalid-token", "nova-senha-segura")
	if !errors.Is(err, auth.ErrInvalidPasswordRecovery) || store.called {
		t.Fatalf("error = %v, store called = %t", err, store.called)
	}
}

func TestPasswordRecoveryVerificationRejectsInvalidPasswordBeforeStore(t *testing.T) {
	service, store, _ := validRecoveryVerification()
	err := service.Verify(context.Background(), testOpaqueRecoveryToken(t), "curta")
	if !errors.Is(err, security.ErrInvalidPassword) || store.called {
		t.Fatalf("error = %v, store called = %t", err, store.called)
	}
}

func TestPasswordRecoveryVerificationPreparesHashForEligibleOwner(t *testing.T) {
	service, store, passwords := validRecoveryVerification()
	token := testOpaqueRecoveryToken(t)
	err := service.Verify(context.Background(), token, "nova-senha-segura")
	if err != nil || !store.called || store.passwordHash != "new-credential-hash" || !passwords.verified || !passwords.hashed {
		t.Fatalf("recovery did not prepare the credential: %v", err)
	}
	if !bytes.Equal(store.hash, security.HashOpaqueToken(token)) || bytes.Equal(store.hash, []byte(token)) {
		t.Fatal("store received a usable token")
	}
}

func TestPasswordRecoveryVerificationRejectsIneligibleStatesWithoutCryptography(t *testing.T) {
	cases := map[string]func(*recoveryVerificationStoreStub, time.Time){
		"inactive":        func(s *recoveryVerificationStoreStub, _ time.Time) { s.user.Active = false },
		"unconfirmed":     func(s *recoveryVerificationStoreStub, _ time.Time) { s.user.EmailConfirmed = false },
		"professional":    func(s *recoveryVerificationStoreStub, _ time.Time) { s.user.Role = "professional" },
		"wrong owner":     func(s *recoveryVerificationStoreStub, _ time.Time) { s.token.UserID = "another-owner" },
		"expiry boundary": func(s *recoveryVerificationStoreStub, now time.Time) { s.token.ExpiresAt = now },
		"expired":         func(s *recoveryVerificationStoreStub, now time.Time) { s.token.ExpiresAt = now.Add(-time.Second) },
		"consumed":        func(s *recoveryVerificationStoreStub, _ time.Time) { s.token.Consumed = true },
		"invalidated":     func(s *recoveryVerificationStoreStub, _ time.Time) { s.token.Invalidated = true },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			service, store, passwords := validRecoveryVerification()
			mutate(store, service.now())
			err := service.Verify(context.Background(), testOpaqueRecoveryToken(t), "nova-senha-segura")
			if !errors.Is(err, auth.ErrInvalidPasswordRecovery) || passwords.verified || passwords.hashed || store.passwordHash != "" {
				t.Fatalf("ineligible state was not rejected before cryptography: %v", err)
			}
		})
	}
}

func TestPasswordRecoveryVerificationRequiresDifferentPassword(t *testing.T) {
	service, store, passwords := validRecoveryVerification()
	passwords.unchanged = true
	err := service.Verify(context.Background(), testOpaqueRecoveryToken(t), "nova-senha-segura")
	if !errors.Is(err, auth.ErrPasswordUnchanged) || passwords.hashed || store.passwordHash != "" {
		t.Fatalf("unchanged password was not rejected: %v", err)
	}
}

func TestPasswordRecoveryVerificationPropagatesFailuresWithoutPreparingCredential(t *testing.T) {
	for _, operation := range []string{"lookup", "verify", "hash"} {
		t.Run(operation, func(t *testing.T) {
			service, store, passwords := validRecoveryVerification()
			failure := errors.New("controlled failure")
			switch operation {
			case "lookup":
				store.err = failure
			case "verify":
				passwords.verifyErr = failure
			case "hash":
				passwords.hashErr = failure
			}
			err := service.Verify(context.Background(), testOpaqueRecoveryToken(t), "nova-senha-segura")
			if !errors.Is(err, failure) || store.passwordHash != "" {
				t.Fatalf("failed recovery prepared a credential: %v", err)
			}
		})
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
