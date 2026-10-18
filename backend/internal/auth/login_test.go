package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"
)

var testNow = time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)

type loginStoreStub struct {
	user               User
	findErr            error
	attempts           map[string][]time.Time
	recordedFailures   int
	createdSession     bool
	createdUserID      string
	createdRefreshHash []byte
}

func (stub *loginStoreStub) FindUserByNormalizedEmail(context.Context, string) (User, error) {
	return stub.user, stub.findErr
}

func (stub *loginStoreStub) ListRateLimitEvents(_ context.Context, _ string, dimension string, _ []byte, _ time.Time) ([]time.Time, error) {
	return stub.attempts[dimension], nil
}

func (stub *loginStoreStub) RecordRateLimitEvents(context.Context, string, map[string][]byte, time.Time) error {
	stub.recordedFailures++
	return nil
}

func (stub *loginStoreStub) CreateSession(_ context.Context, userID string, refreshHash []byte, _ time.Time) error {
	stub.createdSession = true
	stub.createdUserID = userID
	stub.createdRefreshHash = append([]byte(nil), refreshHash...)
	return nil
}

type passwordVerifierStub struct {
	match bool
}

func (stub passwordVerifierStub) Verify(string, string) (bool, error) { return stub.match, nil }

type accessIssuerStub struct{}

func (accessIssuerStub) Issue(principal security.Principal, _ time.Time) (string, error) {
	if principal.UserID == "" {
		return "", errors.New("missing user")
	}
	return "access-token", nil
}

type signedIssuerStub struct{}

func (signedIssuerStub) NewSignedToken() (string, error) { return "csrf-token", nil }

type rateKeyStub struct{}

func (rateKeyStub) Key(dimension, _ string) []byte { return []byte(dimension) }

func TestLoginCreatesSessionForConfirmedActiveUser(t *testing.T) {
	store := &loginStoreStub{user: validLoginUser(), attempts: map[string][]time.Time{}}
	service := testLoginService(store, true)

	result, err := service.Login(context.Background(), LoginInput{Email: " USER@EXAMPLE.COM ", Password: "senha-segura", IP: "192.0.2.1"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !store.createdSession || store.createdUserID != "user-1" || len(store.createdRefreshHash) == 0 {
		t.Fatalf("session was not created correctly: %#v", store)
	}
	if result.AccessToken != "access-token" || result.CSRFToken != "csrf-token" || result.RefreshToken == "" {
		t.Fatalf("unexpected login result: %#v", result)
	}
	if result.Principal.Email != "user@example.com" {
		t.Fatalf("principal email = %q", result.Principal.Email)
	}
}

func TestLoginUsesGenericFailureForUnknownInvalidOrInactiveUser(t *testing.T) {
	tests := []struct {
		name  string
		store *loginStoreStub
		match bool
	}{
		{name: "unknown", store: &loginStoreStub{findErr: ErrUserNotFound, attempts: map[string][]time.Time{}}, match: false},
		{name: "wrong password", store: &loginStoreStub{user: validLoginUser(), attempts: map[string][]time.Time{}}, match: false},
		{name: "inactive", store: &loginStoreStub{user: func() User { user := validLoginUser(); user.Active = false; return user }(), attempts: map[string][]time.Time{}}, match: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := testLoginService(test.store, test.match).Login(context.Background(), LoginInput{Email: "user@example.com", Password: "senha", IP: "192.0.2.1"})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("error = %v, want ErrInvalidCredentials", err)
			}
			if test.store.recordedFailures != 1 || test.store.createdSession {
				t.Fatalf("unexpected store effects: %#v", test.store)
			}
		})
	}
}

func TestLoginReturnsConfirmationStateOnlyAfterCorrectPassword(t *testing.T) {
	store := &loginStoreStub{user: func() User { user := validLoginUser(); user.EmailConfirmed = false; return user }(), attempts: map[string][]time.Time{}}
	_, err := testLoginService(store, true).Login(context.Background(), LoginInput{Email: "user@example.com", Password: "senha", IP: "192.0.2.1"})
	if !errors.Is(err, ErrEmailConfirmationRequired) {
		t.Fatalf("error = %v, want ErrEmailConfirmationRequired", err)
	}
	if store.createdSession || store.recordedFailures != 0 {
		t.Fatalf("unexpected store effects: %#v", store)
	}
}

func TestLoginAppliesProgressiveRateLimitBeforeCheckingCredentials(t *testing.T) {
	attempts := []time.Time{
		testNow.Add(-time.Minute), testNow.Add(-2 * time.Minute), testNow.Add(-3 * time.Minute),
		testNow.Add(-4 * time.Minute), testNow.Add(-5 * time.Minute),
	}
	store := &loginStoreStub{user: validLoginUser(), attempts: map[string][]time.Time{"ip": attempts}}
	_, err := testLoginService(store, true).Login(context.Background(), LoginInput{Email: "user@example.com", Password: "senha", IP: "192.0.2.1"})
	var rateError RateLimitError
	if !errors.As(err, &rateError) {
		t.Fatalf("error = %v, want RateLimitError", err)
	}
	if rateError.RetryAfter != 10*time.Minute {
		t.Fatalf("RetryAfter = %s, want 10m", rateError.RetryAfter)
	}
	if store.createdSession || store.recordedFailures != 0 {
		t.Fatal("rate-limited login reached credential processing")
	}
}

func TestRateKeysDoNotExposeRawIdentifiers(t *testing.T) {
	signer, err := security.NewHMACSigner(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	if bytes.Contains(signer.Key("email", "sensitive@example.com"), []byte("sensitive@example.com")) {
		t.Fatal("rate key exposes normalized email")
	}
}

func testLoginService(store LoginStore, passwordMatches bool) *LoginService {
	service := NewLoginService(store, passwordVerifierStub{match: passwordMatches}, accessIssuerStub{}, signedIssuerStub{}, rateKeyStub{}, "dummy-hash")
	service.now = func() time.Time { return testNow }
	return service
}

func validLoginUser() User {
	return User{
		ID: "user-1", EstablishmentID: "shop-1", Email: "user@example.com", NormalizedEmail: "user@example.com",
		Role: "owner", Active: true, EmailConfirmed: true, CredentialVersion: 1, PasswordHash: "stored-hash",
	}
}
