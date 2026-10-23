package professionals

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
)

var (
	ErrForbidden        = errors.New("professional registration forbidden")
	ErrInvalidName      = errors.New("invalid professional name")
	ErrEmailUnavailable = errors.New("professional email unavailable")
)

type CreateInput struct {
	Name              string
	Email             string
	TemporaryPassword string
}

type Professional struct {
	ID     string
	Name   string
	Email  string
	Active bool
}

type Creation struct {
	EstablishmentID string
	Name            string
	Email           string
	NormalizedEmail string
	PasswordHash    string
	CreatedAt       time.Time
}

type Store interface {
	Create(context.Context, Creation) (string, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
}

type WriteEntitlements interface {
	RequireWrite(context.Context, string) error
}

type Service struct {
	store        Store
	passwords    PasswordHasher
	entitlements WriteEntitlements
	now          func() time.Time
}

func NewService(store Store, passwords PasswordHasher, entitlements WriteEntitlements) *Service {
	return &Service{store: store, passwords: passwords, entitlements: entitlements, now: time.Now}
}

func (service *Service) Create(ctx context.Context, principal auth.Principal, input CreateInput) (Professional, error) {
	if principal.Role != "owner" {
		return Professional{}, ErrForbidden
	}
	if !utf8.ValidString(input.Name) || strings.TrimSpace(input.Name) == "" || utf8.RuneCountInString(input.Name) > 120 {
		return Professional{}, ErrInvalidName
	}
	if err := auth.ValidateEmail(input.Email); err != nil {
		return Professional{}, err
	}
	if err := security.ValidatePassword(input.TemporaryPassword); err != nil {
		return Professional{}, err
	}
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Professional{}, err
	}
	passwordHash, err := service.passwords.Hash(input.TemporaryPassword)
	if err != nil {
		return Professional{}, err
	}

	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(input.Email)
	id, err := service.store.Create(ctx, Creation{
		EstablishmentID: principal.EstablishmentID, Name: name, Email: email,
		NormalizedEmail: auth.NormalizeEmail(input.Email), PasswordHash: passwordHash, CreatedAt: service.now().UTC(),
	})
	if err != nil {
		return Professional{}, err
	}
	return Professional{ID: id, Name: name, Email: email, Active: true}, nil
}
