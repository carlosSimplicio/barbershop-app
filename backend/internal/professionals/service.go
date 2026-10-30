package professionals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/pagination"
)

var (
	ErrAlreadyActive    = errors.New("professional already active")
	ErrNotFound         = errors.New("professional not found")
	ErrForbidden        = errors.New("professional operation forbidden")
	ErrInvalidName      = errors.New("invalid professional name")
	ErrEmailUnavailable = errors.New("professional email unavailable")
	ErrInvalidLimit     = errors.New("invalid professional list limit")
	ErrInvalidOffset    = errors.New("invalid professional list offset")
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

type ListResult struct {
	Items    []Professional
	Limit    int32
	Offset   int32
	HasMore  bool
	CanWrite bool
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
	Reactivate(context.Context, string, string, string, time.Time) (Professional, error)
	Deactivate(context.Context, string, string, time.Time) (Professional, error)
	UpdateName(context.Context, string, string, string, time.Time) (Professional, error)
	Get(context.Context, string, string) (Professional, error)
	Create(context.Context, Creation) (string, error)
	List(context.Context, string, int32, int32) ([]Professional, error)
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
	if !IsNameValid(input.Name) {
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
	professionalID, err := service.store.Create(ctx, Creation{
		EstablishmentID: principal.EstablishmentID, Name: name, Email: email,
		NormalizedEmail: auth.NormalizeEmail(input.Email), PasswordHash: passwordHash, CreatedAt: service.now().UTC(),
	})
	if err != nil {
		return Professional{}, err
	}
	return Professional{ID: professionalID, Name: name, Email: email, Active: true}, nil
}

func (service *Service) List(ctx context.Context, principal auth.Principal, limit, offset int32) (ListResult, error) {
	if principal.Role != "owner" {
		return ListResult{}, ErrForbidden
	}
	if limit < pagination.MinLimit || limit > pagination.MaxLimit {
		return ListResult{}, ErrInvalidLimit
	}
	if offset < pagination.MinOffset {
		return ListResult{}, ErrInvalidOffset
	}
	writeErr := service.entitlements.RequireWrite(ctx, principal.EstablishmentID)
	if writeErr != nil && !errors.Is(writeErr, entitlements.ErrReadOnly) {
		return ListResult{}, writeErr
	}
	items, err := service.store.List(ctx, principal.EstablishmentID, limit+1, offset)
	if err != nil {
		return ListResult{}, fmt.Errorf("list professionals: %w", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	if items == nil {
		items = []Professional{}
	}
	return ListResult{Items: items, Limit: limit, Offset: offset, HasMore: hasMore, CanWrite: writeErr == nil}, nil
}

func (service *Service) Get(ctx context.Context, principal auth.Principal, professionalID string) (Professional, error) {
	if principal.Role != "owner" {
		return Professional{}, ErrForbidden
	}
	return service.store.Get(ctx, principal.EstablishmentID, professionalID)
}

func (service *Service) UpdateName(ctx context.Context, principal auth.Principal, professionalID, name string) (Professional, error) {
	if principal.Role != "owner" {
		return Professional{}, ErrForbidden
	}
	if !IsNameValid(name) {
		return Professional{}, ErrInvalidName
	}
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Professional{}, err
	}
	return service.store.UpdateName(ctx, principal.EstablishmentID, professionalID, strings.TrimSpace(name), service.now().UTC())
}

func (service *Service) Deactivate(ctx context.Context, principal auth.Principal, professionalID string) (Professional, error) {
	if principal.Role != "owner" {
		return Professional{}, ErrForbidden
	}
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Professional{}, err
	}
	return service.store.Deactivate(ctx, principal.EstablishmentID, professionalID, service.now().UTC())
}

func (service *Service) Reactivate(ctx context.Context, principal auth.Principal, professionalID, temporaryPassword string) (Professional, error) {
	if principal.Role != "owner" {
		return Professional{}, ErrForbidden
	}
	if err := security.ValidatePassword(temporaryPassword); err != nil {
		return Professional{}, err
	}
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Professional{}, err
	}
	passwordHash, err := service.passwords.Hash(temporaryPassword)
	if err != nil {
		return Professional{}, err
	}
	return service.store.Reactivate(ctx, principal.EstablishmentID, professionalID, passwordHash, service.now().UTC())
}
