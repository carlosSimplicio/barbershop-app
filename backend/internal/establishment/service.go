package establishment

import (
	"context"
	"fmt"
	"time"

	"barbershop-app/backend/internal/auth"
)

type TrialStatus string

const (
	TrialStatusActive  TrialStatus = "active"
	TrialStatusExpired TrialStatus = "expired"
)

type Trial struct {
	Status   TrialStatus
	StartsAt time.Time
	EndsAt   time.Time
}

type Establishment struct {
	ID       string
	Name     string
	Slug     string
	TimeZone string
	Trial    *Trial
}

type Store interface {
	GetByID(context.Context, string) (Establishment, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (service *Service) Get(ctx context.Context, principal auth.Principal) (Establishment, error) {
	establishment, err := service.store.GetByID(ctx, principal.EstablishmentID)
	if err != nil {
		return Establishment{}, fmt.Errorf("get authenticated establishment: %w", err)
	}
	if establishment.Trial == nil {
		return establishment, nil
	}

	establishment.Trial.StartsAt = establishment.Trial.StartsAt.UTC()
	establishment.Trial.EndsAt = establishment.Trial.EndsAt.UTC()
	now := service.now().UTC()
	if !now.Before(establishment.Trial.StartsAt) && now.Before(establishment.Trial.EndsAt) {
		establishment.Trial.Status = TrialStatusActive
	} else {
		establishment.Trial.Status = TrialStatusExpired
	}
	return establishment, nil
}
