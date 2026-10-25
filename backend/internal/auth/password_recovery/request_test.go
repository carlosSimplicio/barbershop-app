package passwordrecovery

import (
	"barbershop-app/backend/internal/auth"
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
)

type recoveryStoreStub struct {
	account     auth.PasswordRecoveryAccount
	found       bool
	token       auth.PasswordRecoveryToken
	replaced    bool
	lookedUp    bool
	replacedFor string
}

func (stub *recoveryStoreStub) FindEligibleOwnerForPasswordRecovery(_ context.Context, _ string) (auth.PasswordRecoveryAccount, bool, error) {
	stub.lookedUp = true
	return stub.account, stub.found, nil
}

func (stub *recoveryStoreStub) ReplacePasswordRecoveryToken(_ context.Context, userID string) (auth.PasswordRecoveryToken, bool, error) {
	stub.replacedFor = userID
	return stub.token, stub.replaced, nil
}

type recoveryRateStoreStub struct {
	denyDimension string
	keys          map[string][]byte
	scope         string
}

func (stub *recoveryRateStoreStub) ReserveRateLimitEvents(_ context.Context, scope string, keys map[string][]byte, _ time.Time) ([]int64, time.Duration, error) {
	stub.scope, stub.keys = scope, keys
	if _, present := keys[stub.denyDimension]; stub.denyDimension != "" && present {
		return nil, time.Minute, nil
	}
	return []int64{1, 2}, 0, nil
}
func (*recoveryRateStoreStub) ReleaseRateLimitEvents(context.Context, []int64) error { return nil }

type recoverySenderStub struct {
	message mailer.Message
	err     error
}

func (stub *recoverySenderStub) Send(_ context.Context, message mailer.Message) error {
	stub.message = message
	return stub.err
}

func newTestRecoveryService(t *testing.T, store *recoveryStoreStub, sender *recoverySenderStub) *PasswordRecoveryRequestService {
	t.Helper()
	keys, err := security.NewHMACSigner(bytes.Repeat([]byte{0x52}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return NewPasswordRecoveryRequestService(store, &recoveryRateStoreStub{}, keys, sender, "https://app.example.test")
}

func TestPasswordRecoveryRequestUsesSeparateIPAndNormalizedEmailRateKeys(t *testing.T) {
	for _, dimension := range []string{"ip", "email"} {
		t.Run(dimension, func(t *testing.T) {
			store := &recoveryStoreStub{}
			sender := &recoverySenderStub{}
			keys, err := security.NewHMACSigner(bytes.Repeat([]byte{0x52}, 32))
			if err != nil {
				t.Fatal(err)
			}
			rates := &recoveryRateStoreStub{denyDimension: dimension}
			service := NewPasswordRecoveryRequestService(store, rates, keys, sender, "https://app.example.test")
			_, err = service.Request(context.Background(), " OWNER@Example.Test ", "192.0.2.1")
			var limited ratelimit.Error
			if !errors.As(err, &limited) || store.lookedUp || rates.scope != "password-recovery" ||
				!bytes.Equal(rates.keys["email"], keys.Key("password-recovery:email", "owner@example.test")) ||
				!bytes.Equal(rates.keys["ip"], keys.Key("password-recovery:ip", "192.0.2.1")) {
				t.Fatalf("error = %v, lookup = %t, scope = %q, keys = %#v", err, store.lookedUp, rates.scope, rates.keys)
			}
		})
	}
}

func TestPasswordRecoveryRequestRejectsInvalidEmailBeforeLookup(t *testing.T) {
	store := &recoveryStoreStub{}
	_, err := newTestRecoveryService(t, store, &recoverySenderStub{}).Request(context.Background(), "invalid", "192.0.2.1")
	if !errors.Is(err, auth.ErrInvalidEmail) || store.lookedUp {
		t.Fatalf("error = %v, lookup = %t", err, store.lookedUp)
	}
}

func TestPasswordRecoveryRequestOnlySendsAfterEligibleReplacement(t *testing.T) {
	for _, test := range []struct {
		name     string
		found    bool
		replaced bool
		wantMail bool
	}{
		{"missing or ineligible", false, false, false},
		{"eligibility lost before lock", true, false, false},
		{"eligible", true, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &recoveryStoreStub{account: auth.PasswordRecoveryAccount{UserID: "user-1", Email: "owner@example.test"}, found: test.found,
				token: auth.PasswordRecoveryToken{Token: "opaque-token"}, replaced: test.replaced}
			sender := &recoverySenderStub{}
			result, err := newTestRecoveryService(t, store, sender).Request(context.Background(), "OWNER@example.test", "192.0.2.1")
			if err != nil || result.DeliveryFailed || (sender.message.To != "") != test.wantMail {
				t.Fatalf("result = %+v, error = %v, mail = %+v", result, err, sender.message)
			}
			if test.wantMail && store.replacedFor != "user-1" {
				t.Fatalf("replaced for %q", store.replacedFor)
			}
		})
	}
}

func TestPasswordRecoveryRequestMasksMailTransportFailure(t *testing.T) {
	store := &recoveryStoreStub{account: auth.PasswordRecoveryAccount{UserID: "user-1", Email: "owner@example.test"}, found: true,
		token: auth.PasswordRecoveryToken{Token: "opaque-token"}, replaced: true}
	sender := &recoverySenderStub{err: errors.New("transport credentials and recipient")}
	result, err := newTestRecoveryService(t, store, sender).Request(context.Background(), "owner@example.test", "192.0.2.1")
	if err != nil || !result.DeliveryFailed {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}
