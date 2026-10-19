package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/ratelimit"
)

var registrationTestNow = time.Date(2026, time.October, 18, 18, 0, 0, 0, time.UTC)

func TestRegistrationCreatesTenantOwnerCredentialAndConfirmationBeforeSending(t *testing.T) {
	store := newRegistrationStoreStub()
	mailer := &registrationMailerStub{}
	service := newTestRegistrationService(store, mailer)

	result, err := service.Register(context.Background(), RegistrationInput{
		EstablishmentName: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo",
		Email: " Owner@Example.COM ", Password: "senha-segura", IP: "192.0.2.10",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.DeliveryFailed {
		t.Fatal("DeliveryFailed = true, want false")
	}
	if store.registration.Email != "Owner@Example.COM" || store.registration.NormalizedEmail != "owner@example.com" {
		t.Fatalf("stored emails = %q/%q", store.registration.Email, store.registration.NormalizedEmail)
	}
	if store.registration.Role != "owner" || store.registration.PasswordHash != "argon2id-hash" {
		t.Fatalf("stored owner = %#v", store.registration)
	}
	if len(store.registration.ConfirmationTokenHash) != 32 || !store.registration.ConfirmationExpiresAt.Equal(registrationTestNow.Add(24*time.Hour)) {
		t.Fatalf("stored confirmation = %x / %s", store.registration.ConfirmationTokenHash, store.registration.ConfirmationExpiresAt)
	}
	if mailer.recipient != "Owner@Example.COM" || mailer.token == "" || !store.createdBeforeSend {
		t.Fatalf("mailer call = recipient %q token %q createdBeforeSend %t", mailer.recipient, mailer.token, store.createdBeforeSend)
	}
	if store.recordedScope != "register" || string(store.recordedKeys["email"]) != "register:email:owner@example.com" || string(store.recordedKeys["ip"]) != "register:ip:192.0.2.10" {
		t.Fatalf("rate limit record = %q %#v", store.recordedScope, store.recordedKeys)
	}
}

func TestRegistrationReturnsSuccessWhenDeliveryFailsAfterPersistence(t *testing.T) {
	store := newRegistrationStoreStub()
	mailer := &registrationMailerStub{err: errors.New("smtp unavailable")}
	service := newTestRegistrationService(store, mailer)

	result, err := service.Register(context.Background(), validRegistrationInput())
	if err != nil || !result.DeliveryFailed || !store.created {
		t.Fatalf("result = %#v, error = %v, created = %t", result, err, store.created)
	}
}

func TestRegistrationRejectsInvalidFieldsBeforePersistence(t *testing.T) {
	tests := []struct {
		name  string
		input RegistrationInput
		want  error
	}{
		{name: "name", input: withRegistrationInput(func(input *RegistrationInput) { input.EstablishmentName = strings.Repeat("a", 121) }), want: ErrInvalidName},
		{name: "slug", input: withRegistrationInput(func(input *RegistrationInput) { input.Slug = "Admin" }), want: ErrInvalidSlug},
		{name: "reserved slug", input: withRegistrationInput(func(input *RegistrationInput) { input.Slug = "admin" }), want: ErrReservedSlug},
		{name: "time zone", input: withRegistrationInput(func(input *RegistrationInput) { input.TimeZone = "Mars/Olympus" }), want: ErrInvalidTimeZone},
		{name: "email", input: withRegistrationInput(func(input *RegistrationInput) { input.Email = "owner" }), want: auth.ErrInvalidEmail},
		{name: "password", input: withRegistrationInput(func(input *RegistrationInput) { input.Password = "short" }), want: security.ErrInvalidPassword},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newRegistrationStoreStub()
			_, err := newTestRegistrationService(store, &registrationMailerStub{}).Register(context.Background(), test.input)
			if !errors.Is(err, test.want) || store.created || store.releasedAttempts != 1 {
				t.Fatalf("error = %v, created = %t, released attempts = %d", err, store.created, store.releasedAttempts)
			}
		})
	}
}

func TestRegistrationReturnsFieldConflictsWithoutSending(t *testing.T) {
	for _, conflict := range []error{ErrEmailUnavailable, ErrSlugUnavailable} {
		store := newRegistrationStoreStub()
		store.createErr = conflict
		mailer := &registrationMailerStub{}
		_, err := newTestRegistrationService(store, mailer).Register(context.Background(), validRegistrationInput())
		if !errors.Is(err, conflict) || mailer.token != "" {
			t.Fatalf("conflict %v: error = %v, token sent = %q", conflict, err, mailer.token)
		}
	}
}

func TestRegistrationHonorsRateLimitBeforeHashingOrPersistence(t *testing.T) {
	store := newRegistrationStoreStub()
	store.events["email"] = []time.Time{
		registrationTestNow.Add(-time.Minute), registrationTestNow.Add(-2 * time.Minute), registrationTestNow.Add(-3 * time.Minute),
		registrationTestNow.Add(-4 * time.Minute), registrationTestNow.Add(-5 * time.Minute),
	}
	hasher := &registrationPasswordHasherStub{}
	service := NewRegistrationService(store, store, hasher, &registrationMailerStub{}, registrationRateKeyStub{})
	service.now = func() time.Time { return registrationTestNow }

	_, err := service.Register(context.Background(), validRegistrationInput())
	var rateError ratelimit.Error
	if !errors.As(err, &rateError) || hasher.called || store.created {
		t.Fatalf("error = %v, hashed = %t, created = %t", err, hasher.called, store.created)
	}
}

func validRegistrationInput() RegistrationInput {
	return RegistrationInput{EstablishmentName: "Barbearia Central", Slug: "barbearia-central", TimeZone: "America/Sao_Paulo", Email: "owner@example.com", Password: "senha-segura", IP: "192.0.2.10"}
}

func withRegistrationInput(change func(*RegistrationInput)) RegistrationInput {
	input := validRegistrationInput()
	change(&input)
	return input
}

func newTestRegistrationService(store *registrationStoreStub, mailer *registrationMailerStub) *RegistrationService {
	service := NewRegistrationService(store, store, &registrationPasswordHasherStub{}, mailer, registrationRateKeyStub{})
	service.now = func() time.Time { return registrationTestNow }
	store.mailer = mailer
	return service
}

type registrationStoreStub struct {
	events            map[string][]time.Time
	registration      Registration
	createErr         error
	created           bool
	mailer            *registrationMailerStub
	createdBeforeSend bool
	recordedScope     string
	recordedKeys      map[string][]byte
	releasedAttempts  int
}

func newRegistrationStoreStub() *registrationStoreStub {
	return &registrationStoreStub{events: make(map[string][]time.Time)}
}

func (stub *registrationStoreStub) CreateRegistration(_ context.Context, registration Registration) error {
	stub.registration = registration
	stub.created = true
	stub.createdBeforeSend = stub.mailer == nil || stub.mailer.token == ""
	return stub.createErr
}

func (stub *registrationStoreStub) ReserveRateLimitEvents(_ context.Context, scope string, keys map[string][]byte, at time.Time) ([]int64, time.Duration, error) {
	stub.recordedScope = scope
	stub.recordedKeys = keys
	for dimension := range keys {
		if len(stub.events[dimension]) >= 5 {
			return nil, stub.events[dimension][4].Add(15 * time.Minute).Sub(at), nil
		}
	}
	return []int64{1, 2}, 0, nil
}

func (stub *registrationStoreStub) ReleaseRateLimitEvents(context.Context, []int64) error {
	stub.releasedAttempts++
	return nil
}

type registrationPasswordHasherStub struct{ called bool }

func (stub *registrationPasswordHasherStub) Hash(string) (string, error) {
	stub.called = true
	return "argon2id-hash", nil
}

type registrationMailerStub struct {
	recipient string
	token     string
	err       error
}

func (stub *registrationMailerStub) Send(_ context.Context, recipient, token string) error {
	stub.recipient, stub.token = recipient, token
	return stub.err
}

type registrationRateKeyStub struct{}

func (registrationRateKeyStub) Key(dimension, value string) []byte {
	return []byte(dimension + ":" + value)
}
