package registration

import (
	"context"
	"errors"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/platform/ratelimit"
)

type RegistrationService struct {
	store       onboarding.RegistrationStore
	passwords   onboarding.RegistrationPasswordHasher
	mailer      onboarding.RegistrationMailer
	rateLimiter *ratelimit.Limiter
	now         func() time.Time
}

func NewRegistrationService(store onboarding.RegistrationStore, rateStore ratelimit.Store, passwords onboarding.RegistrationPasswordHasher, mailer onboarding.RegistrationMailer, rateKeys ratelimit.KeyGenerator) *RegistrationService {
	return &RegistrationService{
		store: store, passwords: passwords, mailer: mailer,
		rateLimiter: ratelimit.New(rateStore, rateKeys), now: time.Now,
	}
}

func (service *RegistrationService) Register(ctx context.Context, input onboarding.RegistrationInput) (onboarding.RegistrationResult, error) {
	now := service.now().UTC()
	normalizedEmail := auth.NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	reservation, err := service.rateLimiter.Reserve(ctx, "register", identifiers, now)
	if err != nil {
		return onboarding.RegistrationResult{}, err
	}
	if err := validateRegistration(input); err != nil {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return onboarding.RegistrationResult{}, errors.Join(err, service.rateLimiter.Release(releaseCtx, reservation))
	}
	passwordHash, err := service.passwords.Hash(input.Password)
	if err != nil {
		return onboarding.RegistrationResult{}, err
	}
	confirmation, err := onboarding.NewEmailConfirmation(now)
	if err != nil {
		return onboarding.RegistrationResult{}, err
	}
	email := strings.TrimSpace(input.Email)
	err = service.store.CreateRegistration(ctx, onboarding.Registration{
		EstablishmentName: strings.TrimSpace(input.EstablishmentName), Slug: input.Slug, TimeZone: input.TimeZone,
		Email: email, NormalizedEmail: normalizedEmail, Role: "owner", PasswordHash: passwordHash,
		ConfirmationTokenHash: confirmation.TokenHash, ConfirmationExpiresAt: confirmation.ExpiresAt, CreatedAt: now,
	})
	if err != nil {
		return onboarding.RegistrationResult{}, err
	}
	if err := service.mailer.Send(ctx, email, confirmation.Token); err != nil {
		return onboarding.RegistrationResult{DeliveryFailed: true}, nil
	}
	return onboarding.RegistrationResult{}, nil
}

func validateRegistration(input onboarding.RegistrationInput) error {
	if err := onboarding.ValidateEstablishmentName(input.EstablishmentName); err != nil {
		return err
	}
	if err := onboarding.ValidateSlug(input.Slug); err != nil {
		return err
	}
	if err := onboarding.ValidateTimeZone(input.TimeZone); err != nil {
		return err
	}
	if err := auth.ValidateEmail(input.Email); err != nil {
		return err
	}
	return security.ValidatePassword(input.Password)
}
