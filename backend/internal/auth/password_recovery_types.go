package auth

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidPasswordRecovery = errors.New("invalid password recovery token")

type PasswordRecoveryAccount struct {
	UserID string
	Email  string
}

type PasswordRecoveryToken struct {
	Token     string
	TokenHash []byte
	ExpiresAt time.Time
}

type PasswordRecoveryStore interface {
	FindEligibleOwnerForPasswordRecovery(context.Context, string) (PasswordRecoveryAccount, bool, error)
	ReplacePasswordRecoveryToken(context.Context, string) (PasswordRecoveryToken, bool, error)
}

type PasswordRecoveryRequestResult struct {
	DeliveryFailed bool
}

type PasswordRecoveryState struct {
	UserID      string
	ExpiresAt   time.Time
	Consumed    bool
	Invalidated bool
}

// RecoveredPasswordPreparer runs under the owner and token locks, before any write.
type RecoveredPasswordPreparer func(User, PasswordRecoveryState) (string, error)

type PasswordRecoveryVerificationStore interface {
	ResetOwnerPassword(context.Context, []byte, RecoveredPasswordPreparer) error
}
