package catalog

import (
	"context"
	"errors"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
)

var ErrForbidden = errors.New("catalog operation forbidden")

type Offering struct {
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
	SetStatus(context.Context, string, string, bool) (Offering, error)
	Update(context.Context, string, string, UpdateInput) (Offering, error)
	Get(context.Context, string, string) (Offering, error)
	Create(context.Context, string, string, int32, int32) (Offering, error)
	List(context.Context, string, int32, int32) ([]Offering, error)
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

func (service *Service) Create(ctx context.Context, principal auth.Principal, input CreateInput) (Offering, error) {
	if principal.Role != "owner" {
		return Offering{}, ErrForbidden
	}
	name, err := normalizeName(input.Name)
	if err != nil {
		return Offering{}, err
	}
	if input.PriceCents == nil {
		return Offering{}, ErrInvalidPrice
	}
	if err := validatePrice(*input.PriceCents); err != nil {
		return Offering{}, err
	}
	duration, err := normalizeDuration(input.DurationMinutes)
	if err != nil {
		return Offering{}, err
	}
	if err := service.writes.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Offering{}, err
	}
	return service.store.Create(ctx, principal.EstablishmentID, name, int32(*input.PriceCents), duration)
}

var ErrInvalidLimit = errors.New("invalid offering limit")
var ErrInvalidOffset = errors.New("invalid offering offset")

type ListResult struct {
	Items    []Offering `json:"items"`
	Limit    int64      `json:"limit"`
	Offset   int64      `json:"offset"`
	HasMore  bool       `json:"has_more"`
	CanWrite bool       `json:"can_write"`
}

func (service *Service) List(ctx context.Context, principal auth.Principal, limit, offset int64) (ListResult, error) {
	if principal.Role != "owner" {
		return ListResult{}, ErrForbidden
	}
	if limit < 1 || limit > 100 {
		return ListResult{}, ErrInvalidLimit
	}
	if offset < 0 || offset > 2147483647 {
		return ListResult{}, ErrInvalidOffset
	}
	writeErr := service.writes.RequireWrite(ctx, principal.EstablishmentID)
	if writeErr != nil && !errors.Is(writeErr, entitlements.ErrReadOnly) {
		return ListResult{}, writeErr
	}
	items, err := service.store.List(ctx, principal.EstablishmentID, int32(limit+1), int32(offset))
	if err != nil {
		return ListResult{}, err
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	if items == nil {
		items = []Offering{}
	}
	return ListResult{Items: items, Limit: limit, Offset: offset, HasMore: hasMore, CanWrite: writeErr == nil}, nil
}

var ErrNotFound = errors.New("offering not found")

func (service *Service) Get(ctx context.Context, principal auth.Principal, offeringID string) (Offering, error) {
	if principal.Role != "owner" {
		return Offering{}, ErrForbidden
	}
	return service.store.Get(ctx, principal.EstablishmentID, offeringID)
}

type UpdateInput struct {
	Name            *string
	PriceCents      *int64
	DurationMinutes *int64
	SetDuration     bool
}

func (service *Service) Update(ctx context.Context, principal auth.Principal, offeringID string, input UpdateInput) (Offering, error) {
	if principal.Role != "owner" {
		return Offering{}, ErrForbidden
	}
	if input.Name == nil && input.PriceCents == nil && !input.SetDuration {
		return Offering{}, ErrInvalidRequest
	}
	if input.Name != nil {
		name, err := normalizeName(*input.Name)
		if err != nil {
			return Offering{}, err
		}
		input.Name = &name
	}
	if input.PriceCents != nil {
		if err := validatePrice(*input.PriceCents); err != nil {
			return Offering{}, err
		}
	}
	if input.SetDuration {
		duration, err := normalizeDuration(input.DurationMinutes)
		if err != nil {
			return Offering{}, err
		}
		normalizedDuration := int64(duration)
		input.DurationMinutes = &normalizedDuration
	}
	if err := service.writes.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Offering{}, err
	}
	return service.store.Update(ctx, principal.EstablishmentID, offeringID, input)
}

var ErrInvalidRequest = errors.New("invalid offering request")

var ErrInvalidStatus = errors.New("invalid offering status")

func (service *Service) SetStatus(ctx context.Context, principal auth.Principal, offeringID string, active *bool) (Offering, error) {
	if principal.Role != "owner" {
		return Offering{}, ErrForbidden
	}
	if active == nil {
		return Offering{}, ErrInvalidStatus
	}
	if err := service.writes.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Offering{}, err
	}
	return service.store.SetStatus(ctx, principal.EstablishmentID, offeringID, *active)
}
