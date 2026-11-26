package offerings

import (
	"context"
	"errors"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/catalog"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/pagination"
)

var (
	ErrInvalidRequest = errors.New("invalid professional services request")
	ErrForbidden      = errors.New("professional services forbidden")
	ErrNotFound       = errors.New("professional or service not found")
	ErrInvalidLimit   = errors.New("invalid professional services limit")
	ErrInvalidOffset  = errors.New("invalid professional services offset")
)

type Professional struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Offering struct {
	catalog.Offering
	Associated bool `json:"associated"`
}

type Page struct {
	Professional Professional `json:"professional"`
	Items        []Offering   `json:"items"`
	Limit        int64        `json:"limit"`
	Offset       int64        `json:"offset"`
	HasMore      bool         `json:"has_more"`
	CanWrite     bool         `json:"can_write"`
}

type Store interface {
	Set(context.Context, string, string, string, bool) (Offering, error)
	GetProfessional(context.Context, string, string) (Professional, error)
	List(context.Context, string, string, int32, int32) ([]Offering, error)
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

func (service *Service) List(ctx context.Context, principal auth.Principal, professionalID string, limit, offset int64) (Page, error) {
	if principal.Role != "owner" {
		return Page{}, ErrForbidden
	}
	if limit < pagination.MinLimit || limit > pagination.MaxLimit {
		return Page{}, ErrInvalidLimit
	}
	if offset < pagination.MinOffset || offset > pagination.MaxOffset {
		return Page{}, ErrInvalidOffset
	}
	professional, err := service.store.GetProfessional(ctx, principal.EstablishmentID, professionalID)
	if err != nil {
		return Page{}, err
	}
	writeErr := service.writes.RequireWrite(ctx, principal.EstablishmentID)
	if writeErr != nil && !errors.Is(writeErr, entitlements.ErrReadOnly) {
		return Page{}, writeErr
	}
	items, err := service.store.List(ctx, principal.EstablishmentID, professionalID, int32(limit+1), int32(offset))
	if err != nil {
		return Page{}, err
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	if items == nil {
		items = []Offering{}
	}
	return Page{Professional: professional, Items: items, Limit: limit, Offset: offset, HasMore: hasMore, CanWrite: writeErr == nil}, nil
}

func (service *Service) Set(ctx context.Context, principal auth.Principal, professionalID, offeringID string, associated *bool) (Offering, error) {
	if principal.Role != "owner" {
		return Offering{}, ErrForbidden
	}
	if associated == nil {
		return Offering{}, ErrInvalidRequest
	}
	if err := service.writes.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return Offering{}, err
	}
	return service.store.Set(ctx, principal.EstablishmentID, professionalID, offeringID, *associated)
}
