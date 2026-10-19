package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

var ErrInvalidEmail = errors.New("invalid email")

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || !utf8.ValidString(trimmed) || len(trimmed) > 254 {
		return ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed.Address != trimmed {
		return ErrInvalidEmail
	}
	return nil
}
