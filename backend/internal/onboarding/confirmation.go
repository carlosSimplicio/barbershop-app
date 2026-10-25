package onboarding

import (
	"time"

	"barbershop-app/backend/internal/auth/security"
)

const emailConfirmationLifetime = 24 * time.Hour

func NewEmailConfirmation(now time.Time) (EmailConfirmation, error) {
	token, err := security.NewOpaqueToken()
	if err != nil {
		return EmailConfirmation{}, err
	}
	return EmailConfirmation{Token: token, TokenHash: security.HashOpaqueToken(token), ExpiresAt: now.UTC().Add(emailConfirmationLifetime)}, nil
}
