package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"
)

type sessionStoreStub struct {
	user             User
	findErr          error
	revoked          bool
	revokedSessionID string
	revokedUserID    string
	changed          bool
	expectedVersion  int64
	newPasswordHash  string
	newSessionID     string
	newExpiresAt     time.Time
}

func (stub *sessionStoreStub) FindUserByID(context.Context, string) (User, error) {
	return stub.user, stub.findErr
}

func (stub *sessionStoreStub) RevokeSession(_ context.Context, sessionID, userID string, _ time.Time) error {
	stub.revoked = true
	stub.revokedSessionID = sessionID
	stub.revokedUserID = userID
	return nil
}

func (stub *sessionStoreStub) ChangeTemporaryPassword(_ context.Context, _ string, expectedVersion int64, passwordHash, sessionID string, _, expiresAt time.Time) error {
	stub.changed = true
	stub.expectedVersion = expectedVersion
	stub.newPasswordHash = passwordHash
	stub.newSessionID = sessionID
	stub.newExpiresAt = expiresAt
	return nil
}

type passwordManagerStub struct {
	match bool
}

func (stub passwordManagerStub) Verify(string, string) (bool, error) { return stub.match, nil }
func (stub passwordManagerStub) Hash(string) (string, error)         { return "new-password-hash", nil }

func TestLogoutWithoutTokenIsIdempotent(t *testing.T) {
	store := &sessionStoreStub{}
	if err := testSessionService(store, false).Logout(context.Background(), ""); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if store.revoked {
		t.Fatal("empty logout should not call persistence")
	}
}

func TestLogoutRevokesOnlySessionIdentifiedByAccessToken(t *testing.T) {
	store := &sessionStoreStub{}
	service := NewSessionService(store, passwordManagerStub{}, accessIssuerStub{claims: security.AccessClaims{
		Subject: "user-1", SessionID: "session-1",
	}}, signedIssuerStub{})
	service.now = func() time.Time { return testNow }
	if err := service.Logout(context.Background(), "valid-access"); err != nil {
		t.Fatal(err)
	}
	if !store.revoked || store.revokedSessionID != "session-1" || store.revokedUserID != "user-1" {
		t.Fatalf("unexpected revocation: %#v", store)
	}
}

func TestLogoutWithInvalidAccessIsIdempotent(t *testing.T) {
	store := &sessionStoreStub{}
	service := NewSessionService(store, passwordManagerStub{}, accessIssuerStub{verifyErr: security.ErrInvalidAccessToken}, signedIssuerStub{})
	if err := service.Logout(context.Background(), "invalid-access"); err != nil || store.revoked {
		t.Fatalf("logout error = %v, revoked = %t", err, store.revoked)
	}
}

func TestTemporaryPasswordChangeRevokesOldCredentialAndReissuesSession(t *testing.T) {
	user := validSessionUser()
	user.PasswordChangeRequired = true
	store := &sessionStoreStub{user: user}
	service := testSessionService(store, false)

	result, err := service.ChangeTemporaryPassword(context.Background(), Principal{UserID: user.ID}, "nova-senha-segura")
	if err != nil {
		t.Fatalf("change temporary password: %v", err)
	}
	if !store.changed || store.expectedVersion != 3 || store.newPasswordHash != "new-password-hash" || store.newSessionID == "" || !store.newExpiresAt.Equal(testNow.Add(15*24*time.Hour)) {
		t.Fatalf("unexpected persistence call: %#v", store)
	}
	if result.Principal.PasswordChangeRequired || result.AccessToken == "" || result.CSRFToken == "" {
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

func TestTemporaryPasswordChangePreservesStorageFailure(t *testing.T) {
	storageFailure := errors.New("database unavailable")
	store := &sessionStoreStub{findErr: storageFailure}
	_, err := testSessionService(store, false).ChangeTemporaryPassword(context.Background(), Principal{UserID: "user-1"}, "nova-senha-segura")
	if !errors.Is(err, storageFailure) {
		t.Fatalf("error = %v, want storage failure", err)
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
