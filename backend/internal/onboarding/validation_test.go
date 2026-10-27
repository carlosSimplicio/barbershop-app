package onboarding

import (
	"strings"
	"testing"
)

func TestValidateEstablishmentNameRequiresAtMost120Characters(t *testing.T) {
	for _, name := range []string{"", "   ", strings.Repeat("á", 121)} {
		if err := ValidateEstablishmentName(name); err == nil {
			t.Errorf("ValidateEstablishmentName(%q) = nil, want error", name)
		}
	}
	for _, name := range []string{"Barbearia Central", strings.Repeat("á", 120)} {
		if err := ValidateEstablishmentName(name); err != nil {
			t.Errorf("ValidateEstablishmentName(%q) = %v, want nil", name, err)
		}
	}
}

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
		"Barbearia", "barbearia_central", "-barbearia", "barbearia-", "admin", "api", "assets",
		"cadastro", "login", "confirmar-email", "recuperar-senha", "redefinir-senha",
		"painel", "testes", "design-system",
	}
	for _, slug := range values {
		if err := ValidateSlug(slug); err == nil {
			t.Errorf("ValidateSlug(%q) = nil, want error", slug)
		}
	}
}

func TestValidateSlugDoesNotReserveNonConflictingRouteNamesByPrefix(t *testing.T) {
	for _, slug := range []string{"api-v1", "assets-barbershop", "painel-central"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", slug, err)
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
