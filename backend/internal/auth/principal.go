package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequirePrincipal returns the identity established by the authentication
// middleware, or aborts the HTTP request with the shared authentication error.
func RequirePrincipal(httpContext *gin.Context) (Principal, bool) {
	principal, ok := PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		httpContext.Abort()
	}
	return principal, ok
}
