package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/httpserver"

	"github.com/gin-gonic/gin"
)

const principalContextKey = "auth_principal"

var publicPaths = map[string]struct{}{
	"/api/v1/health":                         {},
	"/api/v1/auth/register":                  {},
	"/api/v1/auth/email-confirmation":        {},
	"/api/v1/auth/email-confirmation/verify": {},
	"/api/v1/auth/login":                     {},
	"/api/v1/auth/logout":                    {},
}

type AccessTokenVerifier interface {
	Verify(string, time.Time) (security.AccessClaims, error)
}

type SignedTokenVerifier interface {
	VerifySignedToken(string) bool
}

type AuthUserStore interface {
	FindActiveSessionUser(context.Context, string, time.Time) (User, error)
}

type Middleware struct {
	users          AuthUserStore
	accessTokens   AccessTokenVerifier
	csrfTokens     SignedTokenVerifier
	trustedOrigins map[string]struct{}
	now            func() time.Time
}

func NewMiddleware(users AuthUserStore, accessTokens AccessTokenVerifier, csrfTokens SignedTokenVerifier, trustedOrigins []string) *Middleware {
	origins := make(map[string]struct{}, len(trustedOrigins))
	for _, origin := range trustedOrigins {
		origins[origin] = struct{}{}
	}
	return &Middleware{users: users, accessTokens: accessTokens, csrfTokens: csrfTokens, trustedOrigins: origins, now: time.Now}
}

func (middleware *Middleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isUnsafeMethod(c.Request.Method) {
			if _, allowed := middleware.trustedOrigins[c.GetHeader("Origin")]; !allowed {
				writeError(c, http.StatusForbidden, "csrf_failed", "Não foi possível validar a origem da solicitação.")
				c.Abort()
				return
			}
		}
		if requiresCSRF(c.Request.Method, c.Request.URL.Path) && !middleware.validCSRF(c) {
			writeError(c, http.StatusForbidden, "csrf_failed", "Não foi possível validar a solicitação.")
			c.Abort()
			return
		}
		if requiresAccessToken(c.Request.Method, c.Request.URL.Path) {
			principal, ok := middleware.authenticate(c)
			if !ok {
				c.Abort()
				return
			}
			if principal.PasswordChangeRequired && c.Request.URL.Path != "/api/v1/auth/me" && c.Request.URL.Path != "/api/v1/auth/password/change" {
				writeError(c, http.StatusForbidden, "password_change_required", "Defina uma nova senha para continuar.")
				c.Abort()
				return
			}
			c.Set(principalContextKey, principal)
		}
		c.Next()
	}
}

func (middleware *Middleware) validCSRF(c *gin.Context) bool {
	cookie, err := c.Cookie(csrfCookieName)
	if err != nil {
		return false
	}
	header := c.GetHeader("X-CSRF-Token")
	return header != "" && subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) == 1 && middleware.csrfTokens.VerifySignedToken(cookie)
}

func (middleware *Middleware) authenticate(c *gin.Context) (Principal, bool) {
	token, err := c.Cookie(accessCookieName)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	now := middleware.now().UTC()
	claims, err := middleware.accessTokens.Verify(token, now)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	user, err := middleware.users.FindActiveSessionUser(c.Request.Context(), claims.SessionID, now)
	if err != nil && !errors.Is(err, ErrInvalidSession) {
		slog.Error("authentication_session_lookup_failed", "request_id", httpserver.RequestID(c), "error", err)
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return Principal{}, false
	}
	if err != nil || !userCanAuthenticate(user) || user.ID != claims.Subject || user.EstablishmentID != claims.EstablishmentID ||
		user.Role != claims.Role || user.CredentialVersion != claims.CredentialVersion {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	return Principal{
		UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email,
		Role: user.Role, PasswordChangeRequired: user.PasswordChangeRequired,
	}, true
}

func PrincipalFromContext(c *gin.Context) (Principal, bool) {
	value, exists := c.Get(principalContextKey)
	principal, ok := value.(Principal)
	return principal, exists && ok
}

func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func requiresCSRF(method, path string) bool {
	if !isUnsafeMethod(method) {
		return false
	}
	return path == "/api/v1/auth/logout" || path == "/api/v1/auth/password/change"
}

func requiresAccessToken(method, path string) bool {
	if _, public := publicPaths[path]; public {
		return false
	}
	return strings.HasPrefix(path, "/api/v1/")
}
