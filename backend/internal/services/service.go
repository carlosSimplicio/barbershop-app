package services

import (
	"barbershop-app/backend/internal/auth"
	"context"
	"errors"
)

var ErrForbidden = errors.New("service operation forbidden")

type Record struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PriceCents      int64  `json:"price_cents"`
	DurationMinutes int64  `json:"duration_minutes"`
	Active          bool   `json:"active"`
}
type CreateInput struct {
	Name            string
	PriceCents      *int64
	DurationMinutes *int64
}
type Store interface {
	Create(context.Context, string, string, int32, int32) (Record, error)
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
func (s *Service) Create(ctx context.Context, p auth.Principal, in CreateInput) (Record, error) {
	if p.Role != "owner" {
		return Record{}, ErrForbidden
	}
	name, err := normalizeName(in.Name)
	if err != nil {
		return Record{}, err
	}
	if in.PriceCents == nil {
		return Record{}, ErrInvalidPrice
	}
	if err := validatePrice(*in.PriceCents); err != nil {
		return Record{}, err
	}
	duration, err := normalizeDuration(in.DurationMinutes)
	if err != nil {
		return Record{}, err
	}
	if err := s.writes.RequireWrite(ctx, p.EstablishmentID); err != nil {
		return Record{}, err
	}
	return s.store.Create(ctx, p.EstablishmentID, name, int32(*in.PriceCents), duration)
}
