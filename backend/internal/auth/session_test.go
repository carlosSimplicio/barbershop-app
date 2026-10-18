package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type sessionStoreStub struct {
	user                 User
	findErr              error
	rotateErr            error
	revoked              bool
	changed              bool
	expectedVersion      int64
	newPasswordHash      string
	newCredentialVersion int64
}

func (stub *sessionStoreStub) FindUserByID(context.Context, string) (User, error) {
	return stub.user, stub.findErr
}

func (stub *sessionStoreStub) RotateRefreshToken(context.Context, []byte, []byte, time.Time, time.Time) (User, error) {
	return stub.user, stub.rotateErr
}

func (stub *sessionStoreStub) RevokeSessionByRefreshToken(context.Context, []byte, time.Time) error {
	stub.revoked = true
	return nil
}

func (stub *sessionStoreStub) ChangeTemporaryPassword(_ context.Context, _ string, expectedVersion int64, passwordHash string, _ []byte, _, _ time.Time) (int64, error) {
	stub.changed = true
	stub.expectedVersion = expectedVersion
	stub.newPasswordHash = passwordHash
	return stub.newCredentialVersion, nil
}

type passwordManagerStub struct {
	match bool
}

func (stub passwordManagerStub) Verify(string, string) (bool, error) { return stub.match, nil }
func (stub passwordManagerStub) Hash(string) (string, error)         { return "new-password-hash", nil }

func TestRefreshRotatesTokenAndReissuesSession(t *testing.T) {
	store := &sessionStoreStub{user: validSessionUser()}
	service := testSessionService(store, false)

	result, err := service.Refresh(context.Background(), "current-refresh-token")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if result.RefreshToken == "" || result.RefreshToken == "current-refresh-token" || result.AccessToken != "access-token" || result.CSRFToken != "csrf-token" {
		t.Fatalf("unexpected refresh result: %#v", result)
	}
}

func TestRefreshPreservesReuseDetectionError(t *testing.T) {
	store := &sessionStoreStub{user: validSessionUser(), rotateErr: ErrRefreshTokenReused}
	_, err := testSessionService(store, false).Refresh(context.Background(), "reused-token")
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Fatalf("error = %v, want ErrRefreshTokenReused", err)
	}
}

func TestRefreshRejectsInactiveUser(t *testing.T) {
	user := validSessionUser()
	user.Active = false
	store := &sessionStoreStub{user: user}
	_, err := testSessionService(store, false).Refresh(context.Background(), "refresh-token")
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("error = %v, want ErrInvalidSession", err)
	}
}

func TestLogoutWithoutTokenIsIdempotent(t *testing.T) {
	store := &sessionStoreStub{}
	if err := testSessionService(store, false).Logout(context.Background(), ""); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if store.revoked {
		t.Fatal("empty logout should not call persistence")
	}
}

func TestTemporaryPasswordChangeRevokesOldCredentialAndReissuesSession(t *testing.T) {
	user := validSessionUser()
	user.PasswordChangeRequired = true
	store := &sessionStoreStub{user: user, newCredentialVersion: 4}
	service := testSessionService(store, false)

	result, err := service.ChangeTemporaryPassword(context.Background(), Principal{UserID: user.ID}, "nova-senha-segura")
	if err != nil {
		t.Fatalf("change temporary password: %v", err)
	}
	if !store.changed || store.expectedVersion != 3 || store.newPasswordHash != "new-password-hash" {
		t.Fatalf("unexpected persistence call: %#v", store)
	}
	if result.Principal.PasswordChangeRequired || result.RefreshToken == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestTemporaryPasswordChangeRejectsSamePassword(t *testing.T) {
	user := validSessionUser()
	user.PasswordChangeRequired = true
	store := &sessionStoreStub{user: user}
	_, err := testSessionService(store, true).ChangeTemporaryPassword(context.Background(), Principal{UserID: user.ID}, "senha-temporaria")
	if !errors.Is(err, ErrPasswordUnchanged) || store.changed {
		t.Fatalf("error = %v, changed = %v", err, store.changed)
	}
}

func testSessionService(store SessionStore, passwordMatches bool) *SessionService {
	service := NewSessionService(store, passwordManagerStub{match: passwordMatches}, accessIssuerStub{}, signedIssuerStub{})
	service.now = func() time.Time { return testNow }
	return service
}

func validSessionUser() User {
	return User{
		ID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: "018f47d2-e886-7f1c-8e8d-8b38eaf16464",
		Email: "user@example.com", NormalizedEmail: "user@example.com", Role: "professional", Active: true,
		EmailConfirmed: true, CredentialVersion: 3, PasswordHash: "temporary-password-hash",
	}
}
