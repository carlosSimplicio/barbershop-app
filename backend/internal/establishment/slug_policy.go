package establishment

import "regexp"

var publicSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$`)

var reservedPublicSlugs = map[string]struct{}{
	"admin": {}, "api": {}, "assets": {},
	"cadastro": {}, "confirmar-email": {}, "design-system": {}, "login": {},
	"painel": {}, "recuperar-senha": {}, "redefinir-senha": {}, "testes": {},
}

// IsPublicSlugReserved reports whether a stored slug conflicts with a platform route.
func IsPublicSlugReserved(slug string) bool {
	_, reserved := reservedPublicSlugs[slug]
	return reserved
}

// IsPublicSlugUsable reports whether a stored slug can identify a public route.
func IsPublicSlugUsable(slug string) bool {
	return IsPublicSlugFormatValid(slug) && !IsPublicSlugReserved(slug)
}

// IsPublicSlugFormatValid reports whether a slug matches the platform route format.
func IsPublicSlugFormatValid(slug string) bool {
	return publicSlugPattern.MatchString(slug)
}
