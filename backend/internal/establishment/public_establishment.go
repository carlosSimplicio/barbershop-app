package establishment

import (
	"context"
	"errors"
	"fmt"
)

var ErrPublicEstablishmentNotFound = errors.New("public establishment not found")

type PublicEstablishment struct {
	ID   string
	Name string
	Slug string
}

type publicEstablishmentStore interface {
	FindPublicEstablishmentBySlug(context.Context, string) (PublicEstablishment, error)
}

// PublicEstablishmentResolver centralizes public slug eligibility and lookup.
type PublicEstablishmentResolver struct {
	store publicEstablishmentStore
}

func NewPublicEstablishmentResolver(store publicEstablishmentStore) *PublicEstablishmentResolver {
	return &PublicEstablishmentResolver{store: store}
}

func (resolver *PublicEstablishmentResolver) Resolve(ctx context.Context, slug string) (PublicEstablishment, error) {
	if slug == "" || IsPublicSlugReserved(slug) {
		return PublicEstablishment{}, ErrPublicEstablishmentNotFound
	}
	establishment, err := resolver.store.FindPublicEstablishmentBySlug(ctx, slug)
	if errors.Is(err, ErrPublicEstablishmentNotFound) {
		return PublicEstablishment{}, ErrPublicEstablishmentNotFound
	}
	if err != nil {
		return PublicEstablishment{}, fmt.Errorf("resolve public establishment slug: %w", err)
	}
	return establishment, nil
}
