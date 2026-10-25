package emailconfirmation

import (
	"barbershop-app/backend/internal/onboarding"
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
)

func TestEmailConfirmationRequestReplacesAndSendsForPendingOwner(t *testing.T) {
	accounts := &confirmationAccountStoreStub{account: onboarding.ConfirmationAccount{UserID: "user-1", Email: "Owner@Example.COM"}, found: true}
	confirmations := &confirmationRequesterStub{}
	rates := newConfirmationRateStoreStub()
	service := newTestEmailConfirmationRequestService(accounts, confirmations, rates)

	result, err := service.Request(context.Background(), onboarding.EmailConfirmationRequestInput{Email: " Owner@Example.COM ", IP: "192.0.2.20"})
	if err != nil || result.DeliveryFailed {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	if accounts.normalizedEmail != "owner@example.com" || confirmations.userID != "user-1" || confirmations.recipient != "Owner@Example.COM" {
		t.Fatalf("account lookup/send = %#v / %#v", accounts, confirmations)
	}
	if rates.recordedScope != "email-confirmation" || string(rates.recordedKeys["email"]) != "email-confirmation:email:owner@example.com" || string(rates.recordedKeys["ip"]) != "email-confirmation:ip:192.0.2.20" {
		t.Fatalf("rate record = %q %#v", rates.recordedScope, rates.recordedKeys)
	}
}

func TestEmailConfirmationRequestIsIndistinguishableForMissingAndIneligibleAccounts(t *testing.T) {
	for _, name := range []string{"missing", "confirmed"} {
		t.Run(name, func(t *testing.T) {
			accounts := &confirmationAccountStoreStub{found: false}
			confirmations := &confirmationRequesterStub{}
			result, err := newTestEmailConfirmationRequestService(accounts, confirmations, newConfirmationRateStoreStub()).Request(
				context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner@example.com", IP: "192.0.2.20"},
			)
			if err != nil || result.DeliveryFailed || confirmations.called {
				t.Fatalf("result = %#v, error = %v, confirmation called = %t", result, err, confirmations.called)
			}
		})
	}
}

func TestEmailConfirmationRequestTreatsDeliveryFailureAsAcceptedAfterReplacement(t *testing.T) {
	accounts := &confirmationAccountStoreStub{account: onboarding.ConfirmationAccount{UserID: "user-1", Email: "owner@example.com"}, found: true}
	confirmations := &confirmationRequesterStub{err: mailer.ErrDeliveryFailed}

	result, err := newTestEmailConfirmationRequestService(accounts, confirmations, newConfirmationRateStoreStub()).Request(
		context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner@example.com", IP: "192.0.2.20"},
	)
	if err != nil || !result.DeliveryFailed {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
}

func TestEmailConfirmationRequestPropagatesPersistenceFailures(t *testing.T) {
	persistenceError := errors.New("persistence failed")
	t.Run("account lookup", func(t *testing.T) {
		accounts := &confirmationAccountStoreStub{err: persistenceError}
		_, err := newTestEmailConfirmationRequestService(accounts, &confirmationRequesterStub{}, newConfirmationRateStoreStub()).Request(
			context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner@example.com", IP: "192.0.2.20"},
		)
		if !errors.Is(err, persistenceError) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("confirmation replacement", func(t *testing.T) {
		accounts := &confirmationAccountStoreStub{account: onboarding.ConfirmationAccount{UserID: "user-1", Email: "owner@example.com"}, found: true}
		_, err := newTestEmailConfirmationRequestService(accounts, &confirmationRequesterStub{err: persistenceError}, newConfirmationRateStoreStub()).Request(
			context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner@example.com", IP: "192.0.2.20"},
		)
		if !errors.Is(err, persistenceError) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestEmailConfirmationRequestRejectsInvalidEmailWithoutAccountLookup(t *testing.T) {
	accounts := &confirmationAccountStoreStub{}
	_, err := newTestEmailConfirmationRequestService(accounts, &confirmationRequesterStub{}, newConfirmationRateStoreStub()).Request(
		context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner", IP: "192.0.2.20"},
	)
	if !errors.Is(err, auth.ErrInvalidEmail) || accounts.called {
		t.Fatalf("error = %v, account lookup called = %t", err, accounts.called)
	}
}

func TestEmailConfirmationRequestHonorsRateLimitBeforeAccountLookup(t *testing.T) {
	accounts := &confirmationAccountStoreStub{}
	rates := newConfirmationRateStoreStub()
	rates.events["email"] = []time.Time{
		registrationTestNow.Add(-time.Minute), registrationTestNow.Add(-2 * time.Minute), registrationTestNow.Add(-3 * time.Minute),
		registrationTestNow.Add(-4 * time.Minute), registrationTestNow.Add(-5 * time.Minute),
	}
	_, err := newTestEmailConfirmationRequestService(accounts, &confirmationRequesterStub{}, rates).Request(
		context.Background(), onboarding.EmailConfirmationRequestInput{Email: "owner@example.com", IP: "192.0.2.20"},
	)
	var rateError ratelimit.Error
	if !errors.As(err, &rateError) || accounts.called {
		t.Fatalf("error = %v, account lookup called = %t", err, accounts.called)
	}
}

func newTestEmailConfirmationRequestService(accounts onboarding.ConfirmationAccountStore, confirmations onboarding.ConfirmationRequester, rates *confirmationRateStoreStub) *EmailConfirmationRequestService {
	service := NewEmailConfirmationRequestService(accounts, confirmations, rates, confirmationRateKeyStub{})
	service.now = func() time.Time { return registrationTestNow }
	return service
}

type confirmationAccountStoreStub struct {
	account         onboarding.ConfirmationAccount
	found           bool
	err             error
	called          bool
	normalizedEmail string
}

func (stub *confirmationAccountStoreStub) FindUnconfirmedOwnerByNormalizedEmail(_ context.Context, normalizedEmail string) (onboarding.ConfirmationAccount, bool, error) {
	stub.called = true
	stub.normalizedEmail = normalizedEmail
	return stub.account, stub.found, stub.err
}

type confirmationRequesterStub struct {
	called    bool
	userID    string
	recipient string
	err       error
}

func (stub *confirmationRequesterStub) Request(_ context.Context, userID, recipient string) error {
	stub.called = true
	stub.userID, stub.recipient = userID, recipient
	return stub.err
}

type confirmationRateStoreStub struct {
	events        map[string][]time.Time
	recordedScope string
	recordedKeys  map[string][]byte
}

func newConfirmationRateStoreStub() *confirmationRateStoreStub {
	return &confirmationRateStoreStub{events: make(map[string][]time.Time)}
}

func (stub *confirmationRateStoreStub) ReserveRateLimitEvents(_ context.Context, scope string, keys map[string][]byte, at time.Time) ([]int64, time.Duration, error) {
	stub.recordedScope = scope
	stub.recordedKeys = keys
	for dimension := range keys {
		if len(stub.events[dimension]) >= 5 {
			return nil, stub.events[dimension][4].Add(15 * time.Minute).Sub(at), nil
		}
	}
	return []int64{1, 2}, 0, nil
}

func (*confirmationRateStoreStub) ReleaseRateLimitEvents(context.Context, []int64) error { return nil }

type confirmationRateKeyStub struct{}

func (confirmationRateKeyStub) Key(dimension, value string) []byte {
	return []byte(dimension + ":" + value)
}
