package onboarding

import (
	"context"
	"errors"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/ratelimit"
)

var (
	ErrEmailUnavailable = errors.New("email unavailable")
	ErrSlugUnavailable  = errors.New("slug unavailable")
)

type RegistrationInput struct {
	EstablishmentName string
	Slug              string
	TimeZone          string
	Email             string
	Password          string
	IP                string
}

type Registration struct {
	EstablishmentName     string
	Slug                  string
	TimeZone              string
	Email                 string
	NormalizedEmail       string
	Role                  string
	PasswordHash          string
	ConfirmationTokenHash []byte
	ConfirmationExpiresAt time.Time
	CreatedAt             time.Time
}

type RegistrationResult struct {
	DeliveryFailed bool
}

type RegistrationStore interface {
	CreateRegistration(context.Context, Registration) error
}

type RegistrationPasswordHasher interface {
	Hash(string) (string, error)
}

type RegistrationMailer interface {
	Send(context.Context, string, string) error
}

type RegistrationService struct {
	store       RegistrationStore
	passwords   RegistrationPasswordHasher
	mailer      RegistrationMailer
	rateLimiter *ratelimit.Limiter
	now         func() time.Time
}

func NewRegistrationService(store RegistrationStore, rateStore ratelimit.Store, passwords RegistrationPasswordHasher, mailer RegistrationMailer, rateKeys ratelimit.KeyGenerator) *RegistrationService {
	return &RegistrationService{
		store: store, passwords: passwords, mailer: mailer,
		rateLimiter: ratelimit.New(rateStore, rateKeys), now: time.Now,
	}
}

func (service *RegistrationService) Register(ctx context.Context, input RegistrationInput) (RegistrationResult, error) {
	now := service.now().UTC()
	normalizedEmail := auth.NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	if err := service.rateLimiter.Check(ctx, "register", identifiers, now); err != nil {
		return RegistrationResult{}, err
	}
	if err := validateRegistration(input); err != nil {
		return RegistrationResult{}, err
	}
	if err := service.rateLimiter.Record(ctx, "register", identifiers, now); err != nil {
		return RegistrationResult{}, err
	}
	passwordHash, err := service.passwords.Hash(input.Password)
	if err != nil {
		return RegistrationResult{}, err
	}
	confirmation, err := NewEmailConfirmation(now)
	if err != nil {
		return RegistrationResult{}, err
	}
	email := strings.TrimSpace(input.Email)
	err = service.store.CreateRegistration(ctx, Registration{
		EstablishmentName: strings.TrimSpace(input.EstablishmentName), Slug: input.Slug, TimeZone: input.TimeZone,
		Email: email, NormalizedEmail: normalizedEmail, Role: "owner", PasswordHash: passwordHash,
		ConfirmationTokenHash: confirmation.TokenHash, ConfirmationExpiresAt: confirmation.ExpiresAt, CreatedAt: now,
	})
	if err != nil {
		return RegistrationResult{}, err
	}
	if err := service.mailer.Send(ctx, email, confirmation.Token); err != nil {
		return RegistrationResult{DeliveryFailed: true}, nil
	}
	return RegistrationResult{}, nil
}

func validateRegistration(input RegistrationInput) error {
	if err := ValidateEstablishmentName(input.EstablishmentName); err != nil {
		return err
	}
	if err := ValidateSlug(input.Slug); err != nil {
		return err
	}
	if err := ValidateTimeZone(input.TimeZone); err != nil {
		return err
	}
	if err := auth.ValidateEmail(input.Email); err != nil {
		return err
	}
	return security.ValidatePassword(input.Password)
}
