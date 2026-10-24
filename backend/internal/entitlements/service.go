package entitlements

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrReadOnly = errors.New("establishment is read-only")

type TrialWindow struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type Store interface {
	GetTrialWindow(context.Context, string) (*TrialWindow, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (service *Service) RequireWrite(ctx context.Context, establishmentID string) error {
	trial, err := service.store.GetTrialWindow(ctx, establishmentID)
	if err != nil {
		return fmt.Errorf("get write entitlement: %w", err)
	}
	if trial == nil {
		return ErrReadOnly
	}

	now := service.now()
	if now.Before(trial.StartsAt) || !now.Before(trial.EndsAt) {
		return ErrReadOnly
	}
	return nil
}
