package onboarding

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEmailUnavailable = errors.New("email unavailable")
	ErrSlugUnavailable  = errors.New("slug unavailable")
)

type RegistrationInput struct {
	EstablishmentName string
	Slug              string
	TimeZone          string
	Email             string
	Password          string
	IP                string
}

type Registration struct {
	EstablishmentName     string
	Slug                  string
	TimeZone              string
	Email                 string
	NormalizedEmail       string
	Role                  string
	PasswordHash          string
	ConfirmationTokenHash []byte
	ConfirmationExpiresAt time.Time
	CreatedAt             time.Time
}

type RegistrationResult struct {
	DeliveryFailed bool
}

type RegistrationStore interface {
	CreateRegistration(context.Context, Registration) error
}

type RegistrationPasswordHasher interface {
	Hash(string) (string, error)
}

type RegistrationMailer interface {
	Send(context.Context, string, string) error
}
