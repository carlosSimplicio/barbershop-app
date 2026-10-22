package auth

import (
	"context"
	"errors"

	"barbershop-app/backend/internal/auth/security"
)

var ErrInvalidPasswordRecovery = errors.New("invalid password recovery token")

type PasswordRecoveryVerificationStore interface {
	ResetOwnerPassword(context.Context, []byte, string, PasswordManager) error
}

type PasswordRecoveryVerificationService struct {
	store     PasswordRecoveryVerificationStore
	passwords PasswordManager
}

func NewPasswordRecoveryVerificationService(store PasswordRecoveryVerificationStore, passwords PasswordManager) *PasswordRecoveryVerificationService {
	return &PasswordRecoveryVerificationService{store: store, passwords: passwords}
}

func (service *PasswordRecoveryVerificationService) Verify(ctx context.Context, token, newPassword string) error {
	if !security.ValidateOpaqueToken(token) {
		return ErrInvalidPasswordRecovery
	}
	if err := security.ValidatePassword(newPassword); err != nil {
		return err
	}
	return service.store.ResetOwnerPassword(ctx, security.HashOpaqueToken(token), newPassword, service.passwords)
}
