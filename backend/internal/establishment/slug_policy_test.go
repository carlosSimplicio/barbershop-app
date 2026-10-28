package establishment

import "testing"

func TestIsPublicSlugReservedRecognizesPlatformRoutes(t *testing.T) {
	reservedSlugs := []string{
		"admin", "api", "assets", "cadastro", "confirmar-email", "design-system", "login",
		"painel", "recuperar-senha", "redefinir-senha", "testes",
	}
	for _, slug := range reservedSlugs {
		if !IsPublicSlugReserved(slug) {
			t.Errorf("IsPublicSlugReserved(%q) = false, want true", slug)
		}
	}
}

func TestIsPublicSlugReservedDoesNotBlockOtherSlugsByPrefix(t *testing.T) {
	for _, slug := range []string{"api-v1", "assets-barbershop", "painel-central", "barbearia-central"} {
		if IsPublicSlugReserved(slug) {
			t.Errorf("IsPublicSlugReserved(%q) = true, want false", slug)
		}
	}
}
