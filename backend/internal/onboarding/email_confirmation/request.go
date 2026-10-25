package emailconfirmation

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
)

type EmailConfirmationRequestService struct {
	accounts      onboarding.ConfirmationAccountStore
	confirmations onboarding.ConfirmationRequester
	rateLimiter   *ratelimit.Limiter
	now           func() time.Time
}

func NewEmailConfirmationRequestService(accounts onboarding.ConfirmationAccountStore, confirmations onboarding.ConfirmationRequester, rateStore ratelimit.Store, rateKeys ratelimit.KeyGenerator) *EmailConfirmationRequestService {
	return &EmailConfirmationRequestService{
		accounts: accounts, confirmations: confirmations, rateLimiter: ratelimit.New(rateStore, rateKeys), now: time.Now,
	}
}

func (service *EmailConfirmationRequestService) Request(ctx context.Context, input onboarding.EmailConfirmationRequestInput) (onboarding.EmailConfirmationRequestResult, error) {
	if err := auth.ValidateEmail(input.Email); err != nil {
		return onboarding.EmailConfirmationRequestResult{}, err
	}
	now := service.now().UTC()
	normalizedEmail := auth.NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	if _, err := service.rateLimiter.Reserve(ctx, "email-confirmation", identifiers, now); err != nil {
		return onboarding.EmailConfirmationRequestResult{}, err
	}
	account, found, err := service.accounts.FindUnconfirmedOwnerByNormalizedEmail(ctx, normalizedEmail)
	if err != nil {
		return onboarding.EmailConfirmationRequestResult{}, err
	}
	if !found {
		return onboarding.EmailConfirmationRequestResult{}, nil
	}
	if err := service.confirmations.Request(ctx, account.UserID, account.Email); err != nil {
		if errors.Is(err, mailer.ErrDeliveryFailed) {
			return onboarding.EmailConfirmationRequestResult{DeliveryFailed: true}, nil
		}
		return onboarding.EmailConfirmationRequestResult{}, err
	}
	return onboarding.EmailConfirmationRequestResult{}, nil
}
