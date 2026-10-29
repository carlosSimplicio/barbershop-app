package emailconfirmation

import (
	"context"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/onboarding"

	"github.com/google/uuid"
)

type EmailConfirmationVerificationService struct {
	store  onboarding.EmailConfirmationVerificationStore
	issuer onboarding.AuthenticationIssuer
	now    func() time.Time
}

func NewEmailConfirmationVerificationService(store onboarding.EmailConfirmationVerificationStore, issuer onboarding.AuthenticationIssuer) *EmailConfirmationVerificationService {
	return &EmailConfirmationVerificationService{store: store, issuer: issuer, now: time.Now}
}

func (service *EmailConfirmationVerificationService) Verify(ctx context.Context, token string) (auth.LoginResult, error) {
	if !security.ValidateOpaqueToken(token) {
		return auth.LoginResult{}, onboarding.ErrInvalidEmailConfirmation
	}
	sessionID := uuid.NewString()
	now := service.now().UTC()
	var result auth.LoginResult
	err := service.store.ConfirmEmailAndCreateSession(
		ctx, security.HashOpaqueToken(token), sessionID, now, now.Add(auth.SessionLifetime),
		func(user auth.User) error {
			var issueErr error
			result, issueErr = service.issuer.Issue(user, sessionID, now)
			return issueErr
		},
	)
	if err != nil {
		return auth.LoginResult{}, err
	}
	return result, nil
}
