package onboarding

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth"
)

var ErrInvalidEmailConfirmation = errors.New("invalid email confirmation")

type EmailConfirmation struct {
	Token     string
	TokenHash []byte
	ExpiresAt time.Time
}

type ConfirmationStore interface {
	ReplaceEmailConfirmation(context.Context, string) (EmailConfirmation, bool, error)
}

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

type EmailConfirmationVerificationStore interface {
	ConfirmEmailAndCreateSession(context.Context, []byte, string, time.Time, time.Time, func(auth.User) error) error
}

type AuthenticationIssuer interface {
	Issue(auth.User, string, time.Time) (auth.LoginResult, error)
}
