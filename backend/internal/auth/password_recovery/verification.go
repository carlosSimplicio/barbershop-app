package passwordrecovery

import (
	"context"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
)

type PasswordRecoveryVerificationService struct {
	store     auth.PasswordRecoveryVerificationStore
	passwords auth.PasswordManager
}

func NewPasswordRecoveryVerificationService(store auth.PasswordRecoveryVerificationStore, passwords auth.PasswordManager) *PasswordRecoveryVerificationService {
	return &PasswordRecoveryVerificationService{store: store, passwords: passwords}
}

func (service *PasswordRecoveryVerificationService) Verify(ctx context.Context, token, newPassword string) error {
	if !security.ValidateOpaqueToken(token) {
		return auth.ErrInvalidPasswordRecovery
	}
	if err := security.ValidatePassword(newPassword); err != nil {
		return err
	}
	return service.store.ResetOwnerPassword(ctx, security.HashOpaqueToken(token), newPassword, service.passwords)
}
