package onboarding

import "testing"

func TestValidateSlugAcceptsCanonicalPublicSlugs(t *testing.T) {
	for _, slug := range []string{"barbearia-central", "abc", "salao123"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", slug, err)
		}
	}
}

func TestValidateSlugRejectsInvalidAndReservedValues(t *testing.T) {
	values := []string{
		"ab", "a234567890123456789012345678901234567890123456789012345678901234",
		"Barbearia", "barbearia_central", "-barbearia", "barbearia-", "admin", "api", "login", "assets",
	}
	for _, slug := range values {
		if err := ValidateSlug(slug); err == nil {
			t.Errorf("ValidateSlug(%q) = nil, want error", slug)
		}
	}
}

func TestValidateTimeZoneAcceptsIANAZoneAndRejectsLocalOrUnknownValues(t *testing.T) {
	for _, zone := range []string{"America/Sao_Paulo", "Europe/Lisbon", "UTC", "Brazil/East"} {
		if err := ValidateTimeZone(zone); err != nil {
			t.Errorf("ValidateTimeZone(%q) = %v, want nil", zone, err)
		}
	}
	for _, zone := range []string{"", "Local", "Mars/Olympus"} {
		if err := ValidateTimeZone(zone); err == nil {
			t.Errorf("ValidateTimeZone(%q) = nil, want error", zone)
		}
	}
}
