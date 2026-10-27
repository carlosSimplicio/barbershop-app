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

type PublicEstablishmentProfile struct {
	Name         string
	Slug         string
	LogoURL      string
	Description  *string
	Address      *string
	ContactPhone *string
	SocialLinks  []SocialLink
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
	if !IsPublicSlugUsable(slug) {
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

type publicEstablishmentProfileStore interface {
	FindPublicEstablishmentProfile(context.Context, string, string) (PublicEstablishmentProfile, error)
}

type PublicEstablishmentReader interface {
	GetPublicEstablishment(context.Context, string) (PublicEstablishmentProfile, error)
}

type PublicEstablishmentService struct {
	resolver *PublicEstablishmentResolver
	store    publicEstablishmentProfileStore
}

func NewPublicEstablishmentService(resolver *PublicEstablishmentResolver, store publicEstablishmentProfileStore) *PublicEstablishmentService {
	return &PublicEstablishmentService{resolver: resolver, store: store}
}

func (service *PublicEstablishmentService) GetPublicEstablishment(ctx context.Context, slug string) (PublicEstablishmentProfile, error) {
	establishment, err := service.resolver.Resolve(ctx, slug)
	if errors.Is(err, ErrPublicEstablishmentNotFound) {
		return PublicEstablishmentProfile{}, ErrPublicEstablishmentNotFound
	}
	if err != nil {
		return PublicEstablishmentProfile{}, err
	}
	profile, err := service.store.FindPublicEstablishmentProfile(ctx, establishment.ID, slug)
	if errors.Is(err, ErrPublicEstablishmentNotFound) {
		return PublicEstablishmentProfile{}, ErrPublicEstablishmentNotFound
	}
	if err != nil {
		return PublicEstablishmentProfile{}, fmt.Errorf("get public establishment profile: %w", err)
	}
	profile.LogoURL = "/api/v1/public/establishments/" + profile.Slug + "/logo"
	if len(profile.SocialLinks) == 0 {
		profile.SocialLinks = nil
	}
	return profile, nil
}
