package schedule

import (
	"context"
	"errors"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("professional schedule not found")
	ErrStalePrincipal = errors.New("professional authentication is no longer current")
)

type Period struct {
	Weekday     int16
	StartMinute int16
	EndMinute   int16
}

type Week struct {
	TimeZone string
	Periods  []Period
	CanWrite bool
}

type Store interface {
	Read(context.Context, string, uuid.UUID) (Week, error)
	CheckTarget(context.Context, string, uuid.UUID) error
	Replace(context.Context, auth.Principal, uuid.UUID, []Period) (Week, error)
}

type WriteEntitlements interface {
	RequireWrite(context.Context, string) error
}

type Service struct {
	store  Store
	writes WriteEntitlements
}

func NewService(store Store, writes WriteEntitlements) *Service {
	return &Service{store: store, writes: writes}
}

func (service *Service) Read(ctx context.Context, principal auth.Principal, professionalID uuid.UUID) (Week, error) {
	if !canAccess(principal, professionalID) {
		return Week{}, ErrNotFound
	}
	week, err := service.store.Read(ctx, principal.EstablishmentID, professionalID)
	if err != nil {
		return Week{}, err
	}
	writeErr := service.writes.RequireWrite(ctx, principal.EstablishmentID)
	if writeErr != nil && !errors.Is(writeErr, entitlements.ErrReadOnly) {
		return Week{}, writeErr
	}
	week.CanWrite = writeErr == nil
	return week, nil
}

func (service *Service) Replace(ctx context.Context, principal auth.Principal, professionalID uuid.UUID, input []periodInput) (Week, error) {
	if !canAccess(principal, professionalID) {
		return Week{}, ErrNotFound
	}
	if err := service.store.CheckTarget(ctx, principal.EstablishmentID, professionalID); err != nil {
		return Week{}, err
	}
	if err := service.writes.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Week{}, err
	}
	periods, err := validatePeriods(input)
	if err != nil {
		return Week{}, err
	}
	week, err := service.store.Replace(ctx, principal, professionalID, periods)
	if err != nil {
		return Week{}, err
	}
	week.CanWrite = true
	return week, nil
}

func canAccess(principal auth.Principal, professionalID uuid.UUID) bool {
	switch principal.Role {
	case "owner":
		return true
	case "professional":
		return principal.UserID == professionalID.String()
	default:
		return false
	}
}
