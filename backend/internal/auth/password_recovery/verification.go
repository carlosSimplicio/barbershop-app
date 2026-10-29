package passwordrecovery

import (
	"context"
	"fmt"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
)

type PasswordRecoveryVerificationService struct {
	store     auth.PasswordRecoveryVerificationStore
	passwords auth.PasswordManager
	now       func() time.Time
}

func NewPasswordRecoveryVerificationService(store auth.PasswordRecoveryVerificationStore, passwords auth.PasswordManager) *PasswordRecoveryVerificationService {
	return &PasswordRecoveryVerificationService{store: store, passwords: passwords, now: time.Now}
}

func (service *PasswordRecoveryVerificationService) Verify(ctx context.Context, token, newPassword string) error {
	if !security.ValidateOpaqueToken(token) {
		return auth.ErrInvalidPasswordRecovery
	}
	if err := security.ValidatePassword(newPassword); err != nil {
		return err
	}
	return service.store.ResetOwnerPassword(ctx, security.HashOpaqueToken(token), func(user auth.User, state auth.PasswordRecoveryState) (string, error) {
		return service.preparePassword(user, state, newPassword)
	})
}

func (service *PasswordRecoveryVerificationService) preparePassword(user auth.User, token auth.PasswordRecoveryState, newPassword string) (string, error) {
	if user.Role != "owner" || !user.Active || !user.EmailConfirmed || token.UserID != user.ID ||
		!token.ExpiresAt.After(service.now().UTC()) || token.Consumed || token.Invalidated {
		return "", auth.ErrInvalidPasswordRecovery
	}
	unchanged, err := service.passwords.Verify(newPassword, user.PasswordHash)
	if err != nil {
		return "", fmt.Errorf("verify previous owner password: %w", err)
	}
	if unchanged {
		return "", auth.ErrPasswordUnchanged
	}
	passwordHash, err := service.passwords.Hash(newPassword)
	if err != nil {
		return "", fmt.Errorf("hash recovered owner password: %w", err)
	}
	return passwordHash, nil
}
