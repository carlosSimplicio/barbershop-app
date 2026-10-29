package auth

import (
	"fmt"

	"barbershop-app/backend/internal/auth/security"
)

func PasswordRequirementsMessage() string {
	return fmt.Sprintf("A senha deve ter de %d a %d caracteres, sem espaços ou caracteres invisíveis.", security.MinPasswordCodePoints, security.MaxPasswordCodePoints)
}
