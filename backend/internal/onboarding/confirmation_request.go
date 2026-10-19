package onboarding

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
)

type ConfirmationAccount struct {
	UserID string
	Email  string
}

type EmailConfirmationRequestInput struct {
	Email string
	IP    string
}

type EmailConfirmationRequestResult struct {
	DeliveryFailed bool
}

type ConfirmationAccountStore interface {
	FindUnconfirmedOwnerByNormalizedEmail(context.Context, string) (ConfirmationAccount, bool, error)
}

type ConfirmationRequester interface {
	Request(context.Context, string, string) error
}

type EmailConfirmationRequestService struct {
	accounts      ConfirmationAccountStore
	confirmations ConfirmationRequester
	rateLimiter   *ratelimit.Limiter
	now           func() time.Time
}

func NewEmailConfirmationRequestService(accounts ConfirmationAccountStore, confirmations ConfirmationRequester, rateStore ratelimit.Store, rateKeys ratelimit.KeyGenerator) *EmailConfirmationRequestService {
	return &EmailConfirmationRequestService{
		accounts: accounts, confirmations: confirmations, rateLimiter: ratelimit.New(rateStore, rateKeys), now: time.Now,
	}
}

func (service *EmailConfirmationRequestService) Request(ctx context.Context, input EmailConfirmationRequestInput) (EmailConfirmationRequestResult, error) {
	if err := auth.ValidateEmail(input.Email); err != nil {
		return EmailConfirmationRequestResult{}, err
	}
	now := service.now().UTC()
	normalizedEmail := auth.NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	if err := service.rateLimiter.Check(ctx, "email-confirmation", identifiers, now); err != nil {
		return EmailConfirmationRequestResult{}, err
	}
	if err := service.rateLimiter.Record(ctx, "email-confirmation", identifiers, now); err != nil {
		return EmailConfirmationRequestResult{}, err
	}
	account, found, err := service.accounts.FindUnconfirmedOwnerByNormalizedEmail(ctx, normalizedEmail)
	if err != nil {
		return EmailConfirmationRequestResult{}, err
	}
	if !found {
		return EmailConfirmationRequestResult{}, nil
	}
	if err := service.confirmations.Request(ctx, account.UserID, account.Email); err != nil {
		if errors.Is(err, mailer.ErrDeliveryFailed) {
			return EmailConfirmationRequestResult{DeliveryFailed: true}, nil
		}
		return EmailConfirmationRequestResult{}, err
	}
	return EmailConfirmationRequestResult{}, nil
}
