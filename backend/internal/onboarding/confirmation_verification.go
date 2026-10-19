package onboarding

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
)

var ErrInvalidEmailConfirmation = errors.New("invalid email confirmation")

type EmailConfirmationVerificationStore interface {
	ConfirmEmailAndCreateSession(context.Context, []byte, []byte, time.Time, time.Time, func(auth.User) error) error
}

type AuthenticationIssuer interface {
	Issue(auth.User, string, time.Time) (auth.LoginResult, error)
}

type EmailConfirmationVerificationService struct {
	store  EmailConfirmationVerificationStore
	issuer AuthenticationIssuer
	now    func() time.Time
}

func NewEmailConfirmationVerificationService(store EmailConfirmationVerificationStore, issuer AuthenticationIssuer) *EmailConfirmationVerificationService {
	return &EmailConfirmationVerificationService{store: store, issuer: issuer, now: time.Now}
}

func (service *EmailConfirmationVerificationService) Verify(ctx context.Context, token string) (auth.LoginResult, error) {
	if !security.ValidateOpaqueToken(token) {
		return auth.LoginResult{}, ErrInvalidEmailConfirmation
	}
	refreshToken, err := security.NewOpaqueToken()
	if err != nil {
		return auth.LoginResult{}, err
	}
	now := service.now().UTC()
	var result auth.LoginResult
	err = service.store.ConfirmEmailAndCreateSession(
		ctx, security.HashOpaqueToken(token), security.HashOpaqueToken(refreshToken), now, now.Add(7*24*time.Hour),
		func(user auth.User) error {
			var issueErr error
			result, issueErr = service.issuer.Issue(user, refreshToken, now)
			return issueErr
		},
	)
	if err != nil {
		return auth.LoginResult{}, err
	}
	return result, nil
}
