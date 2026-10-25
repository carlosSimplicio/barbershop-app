package emailconfirmation

import (
	"barbershop-app/backend/internal/onboarding"
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"github.com/google/uuid"
)

func TestVerifyEmailConfirmationCreatesSessionAndIssuesAuthentication(t *testing.T) {
	token, _ := security.NewOpaqueToken()
	store := &verificationStoreStub{user: auth.User{
		ID: "018f47d2-e886-7f1c-8e8d-8b38eaf16463", EstablishmentID: "018f47d2-e886-7f1c-8e8d-8b38eaf16464",
		Email: "owner@example.com", Role: "owner", Active: true, EmailConfirmed: true, CredentialVersion: 3,
	}}
	issuer := &verificationIssuerStub{result: auth.LoginResult{AccessToken: "access", CSRFToken: "csrf"}}
	service := NewEmailConfirmationVerificationService(store, issuer)
	service.now = func() time.Time { return registrationTestNow }

	result, err := service.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "access" || result.CSRFToken != "csrf" {
		t.Fatalf("result = %#v", result)
	}
	if string(store.tokenHash) != string(security.HashOpaqueToken(token)) || uuid.Validate(store.sessionID) != nil {
		t.Fatal("expected hashed confirmation token and generated session ID")
	}
	if !store.confirmedAt.Equal(registrationTestNow) || !store.expiresAt.Equal(registrationTestNow.Add(15*24*time.Hour)) {
		t.Fatalf("timestamps = %s, %s", store.confirmedAt, store.expiresAt)
	}
	if issuer.sessionID != store.sessionID || !issuer.now.Equal(registrationTestNow) {
		t.Fatal("issuer did not receive persisted session ID")
	}
}

func TestVerifyEmailConfirmationRejectsMalformedTokenWithoutStorage(t *testing.T) {
	store := &verificationStoreStub{}
	service := NewEmailConfirmationVerificationService(store, &verificationIssuerStub{})

	_, err := service.Verify(context.Background(), "malformed")
	if !errors.Is(err, onboarding.ErrInvalidEmailConfirmation) || store.called {
		t.Fatalf("error = %v, store called = %t", err, store.called)
	}
}

func TestVerifyEmailConfirmationPreservesUniformInvalidError(t *testing.T) {
	token, _ := security.NewOpaqueToken()
	store := &verificationStoreStub{err: onboarding.ErrInvalidEmailConfirmation}
	service := NewEmailConfirmationVerificationService(store, &verificationIssuerStub{})

	_, err := service.Verify(context.Background(), token)
	if !errors.Is(err, onboarding.ErrInvalidEmailConfirmation) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyEmailConfirmationReturnsIssuanceFailureAfterPersistence(t *testing.T) {
	token, _ := security.NewOpaqueToken()
	issueFailure := errors.New("access token unavailable")
	service := NewEmailConfirmationVerificationService(&verificationStoreStub{}, &verificationIssuerStub{err: issueFailure})

	_, err := service.Verify(context.Background(), token)
	if !errors.Is(err, issueFailure) {
		t.Fatalf("error = %v", err)
	}
}

type verificationStoreStub struct {
	user        auth.User
	err         error
	called      bool
	tokenHash   []byte
	sessionID   string
	confirmedAt time.Time
	expiresAt   time.Time
}

func (stub *verificationStoreStub) ConfirmEmailAndCreateSession(_ context.Context, tokenHash []byte, sessionID string, confirmedAt, expiresAt time.Time, beforeCommit func(auth.User) error) error {
	stub.called = true
	stub.tokenHash = tokenHash
	stub.sessionID = sessionID
	stub.confirmedAt = confirmedAt
	stub.expiresAt = expiresAt
	if stub.err != nil {
		return stub.err
	}
	return beforeCommit(stub.user)
}

type verificationIssuerStub struct {
	result    auth.LoginResult
	err       error
	sessionID string
	now       time.Time
}

func (stub *verificationIssuerStub) Issue(user auth.User, sessionID string, now time.Time) (auth.LoginResult, error) {
	stub.sessionID = sessionID
	stub.now = now
	stub.result.Principal = auth.Principal{UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email, Role: user.Role}
	return stub.result, stub.err
}
