package establishment

import (
	"context"
	"errors"
	"testing"
)

func TestPublicEstablishmentResolverRejectsReservedSlugBeforeLookup(t *testing.T) {
	store := &publicEstablishmentResolverStore{}
	resolver := NewPublicEstablishmentResolver(store)
	for _, slug := range []string{"api", "-invalid", "invalid-"} {
		if _, err := resolver.Resolve(context.Background(), slug); !errors.Is(err, ErrPublicEstablishmentNotFound) {
			t.Errorf("Resolve(%q) error = %v, want not found", slug, err)
		}
	}
	if store.lookups != 0 {
		t.Fatalf("reserved slug lookup count = %d, want 0", store.lookups)
	}
}

func TestPublicEstablishmentServiceBuildsMinimalPublishedProfile(t *testing.T) {
	store := &publicEstablishmentResolverStore{}
	profileStore := publicEstablishmentProfileStoreStub{profile: PublicEstablishmentProfile{
		Name: "Barbearia Central", Slug: "barbearia-central",
		SocialLinks: []SocialLink{{Label: "Instagram", URL: "https://instagram.com/central"}},
	}}
	service := NewPublicEstablishmentService(NewPublicEstablishmentResolver(store), profileStore)

	profile, err := service.GetPublicEstablishment(context.Background(), "barbearia-central")
	if err != nil {
		t.Fatalf("GetPublicEstablishment() error = %v", err)
	}
	if profile.LogoURL != "/api/v1/public/establishments/barbearia-central/logo" || len(profile.SocialLinks) != 1 {
		t.Fatalf("published profile = %+v", profile)
	}
	if profile.Description != nil || profile.Address != nil || profile.ContactPhone != nil {
		t.Fatalf("absent optional fields should stay absent: %+v", profile)
	}
}

type publicEstablishmentResolverStore struct {
	lookups int
}

func (store *publicEstablishmentResolverStore) FindPublicEstablishmentBySlug(_ context.Context, slug string) (PublicEstablishment, error) {
	store.lookups++
	return PublicEstablishment{ID: testEstablishmentID, Name: "Barbearia Central", Slug: slug}, nil
}

type publicEstablishmentProfileStoreStub struct {
	profile PublicEstablishmentProfile
}

func (store publicEstablishmentProfileStoreStub) FindPublicEstablishmentProfile(context.Context, string, string) (PublicEstablishmentProfile, error) {
	return store.profile, nil
}
