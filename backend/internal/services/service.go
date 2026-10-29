package services

import (
	"context"
	"errors"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
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
	SetStatus(context.Context, string, string, bool) (Record, error)
	Update(context.Context, string, string, UpdateInput) (Record, error)
	Get(context.Context, string, string) (Record, error)
	Create(context.Context, string, string, int32, int32) (Record, error)
	List(context.Context, string, int32, int32) ([]Record, error)
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

var ErrInvalidLimit = errors.New("invalid service limit")
var ErrInvalidOffset = errors.New("invalid service offset")

type ListResult struct {
	Items    []Record `json:"items"`
	Limit    int64    `json:"limit"`
	Offset   int64    `json:"offset"`
	HasMore  bool     `json:"has_more"`
	CanWrite bool     `json:"can_write"`
}

func (s *Service) List(ctx context.Context, p auth.Principal, limit, offset int64) (ListResult, error) {
	if p.Role != "owner" {
		return ListResult{}, ErrForbidden
	}
	if limit < 1 || limit > 100 {
		return ListResult{}, ErrInvalidLimit
	}
	if offset < 0 || offset > 2147483647 {
		return ListResult{}, ErrInvalidOffset
	}
	writeErr := s.writes.RequireWrite(ctx, p.EstablishmentID)
	if writeErr != nil && !errors.Is(writeErr, entitlements.ErrReadOnly) {
		return ListResult{}, writeErr
	}
	items, err := s.store.List(ctx, p.EstablishmentID, int32(limit+1), int32(offset))
	if err != nil {
		return ListResult{}, err
	}
	more := len(items) > int(limit)
	if more {
		items = items[:limit]
	}
	if items == nil {
		items = []Record{}
	}
	return ListResult{Items: items, Limit: limit, Offset: offset, HasMore: more, CanWrite: writeErr == nil}, nil
}

var ErrNotFound = errors.New("service not found")

func (s *Service) Get(ctx context.Context, p auth.Principal, id string) (Record, error) {
	if p.Role != "owner" {
		return Record{}, ErrForbidden
	}
	return s.store.Get(ctx, p.EstablishmentID, id)
}

type UpdateInput struct {
	Name            *string
	PriceCents      *int64
	DurationMinutes *int64
	SetDuration     bool
}

func (s *Service) Update(ctx context.Context, p auth.Principal, id string, in UpdateInput) (Record, error) {
	if p.Role != "owner" {
		return Record{}, ErrForbidden
	}
	if in.Name == nil && in.PriceCents == nil && !in.SetDuration {
		return Record{}, ErrInvalidRequest
	}
	if in.Name != nil {
		name, err := normalizeName(*in.Name)
		if err != nil {
			return Record{}, err
		}
		in.Name = &name
	}
	if in.PriceCents != nil {
		if err := validatePrice(*in.PriceCents); err != nil {
			return Record{}, err
		}
	}
	if in.SetDuration {
		duration, err := normalizeDuration(in.DurationMinutes)
		if err != nil {
			return Record{}, err
		}
		v := int64(duration)
		in.DurationMinutes = &v
	}
	if err := s.writes.RequireWrite(ctx, p.EstablishmentID); err != nil {
		return Record{}, err
	}
	return s.store.Update(ctx, p.EstablishmentID, id, in)
}

var ErrInvalidRequest = errors.New("invalid service request")

var ErrInvalidStatus = errors.New("invalid service status")

func (s *Service) SetStatus(ctx context.Context, p auth.Principal, id string, active *bool) (Record, error) {
	if p.Role != "owner" {
		return Record{}, ErrForbidden
	}
	if active == nil {
		return Record{}, ErrInvalidStatus
	}
	if err := s.writes.RequireWrite(ctx, p.EstablishmentID); err != nil {
		return Record{}, err
	}
	return s.store.SetStatus(ctx, p.EstablishmentID, id, *active)
}
