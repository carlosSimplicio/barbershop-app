package establishment

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
