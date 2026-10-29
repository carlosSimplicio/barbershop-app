package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/httpserver"

	"github.com/gin-gonic/gin"
)

const principalContextKey = "auth_principal"

type pathRule struct {
	method  string
	path    string
	pattern *regexp.Regexp
}

var publicPaths = []pathRule{
	{path: "/api/v1/health"},
	{path: "/api/v1/auth/register"},
	{path: "/api/v1/auth/email-confirmation"},
	{path: "/api/v1/auth/email-confirmation/verify"},
	{path: "/api/v1/auth/password/recovery"},
	{path: "/api/v1/auth/password/recovery/verify"},
	{path: "/api/v1/auth/login"},
	{path: "/api/v1/auth/logout"},
	{
		method:  http.MethodGet,
		pattern: regexp.MustCompile(`^/api/v1/public/establishments/[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$`),
	},
	{
		method:  http.MethodGet,
		pattern: regexp.MustCompile(`^/api/v1/public/establishments/[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])/logo$`),
	},
}

var csrfProtectedPaths = []pathRule{
	{path: "/api/v1/services"},
	{pattern: regexp.MustCompile(`^/api/v1/services/[^/]+(?:/status)?$`)},
	{path: "/api/v1/auth/logout"},
	{path: "/api/v1/auth/password/change"},
	{path: "/api/v1/establishment/settings"},
	{path: "/api/v1/establishment/logo"},
	{path: "/api/v1/professionals"},
	{pattern: regexp.MustCompile(`^/api/v1/professionals/[^/]+(?:/deactivation)?$`)},
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
	return func(httpContext *gin.Context) {
		if isUnsafeMethod(httpContext.Request.Method) {
			if _, allowed := middleware.trustedOrigins[httpContext.GetHeader("Origin")]; !allowed {
				writeError(httpContext, http.StatusForbidden, "csrf_failed", "Não foi possível validar a origem da solicitação.")
				httpContext.Abort()
				return
			}
		}
		if requiresCSRF(httpContext.Request.Method, httpContext.Request.URL.Path) && !middleware.validCSRF(httpContext) {
			writeError(httpContext, http.StatusForbidden, "csrf_failed", "Não foi possível validar a solicitação.")
			httpContext.Abort()
			return
		}
		if requiresAccessToken(httpContext.Request.Method, httpContext.Request.URL.Path) {
			principal, ok := middleware.authenticate(httpContext)
			if !ok {
				httpContext.Abort()
				return
			}
			if principal.PasswordChangeRequired && httpContext.Request.URL.Path != "/api/v1/auth/me" && httpContext.Request.URL.Path != "/api/v1/auth/password/change" {
				writeError(httpContext, http.StatusForbidden, "password_change_required", "Defina uma nova senha para continuar.")
				httpContext.Abort()
				return
			}
			httpContext.Set(principalContextKey, principal)
		}
		httpContext.Next()
	}
}

func (middleware *Middleware) validCSRF(httpContext *gin.Context) bool {
	cookie, err := httpContext.Cookie(csrfCookieName)
	if err != nil {
		return false
	}
	header := httpContext.GetHeader("X-CSRF-Token")
	return header != "" && subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) == 1 && middleware.csrfTokens.VerifySignedToken(cookie)
}

func (middleware *Middleware) authenticate(httpContext *gin.Context) (Principal, bool) {
	token, err := httpContext.Cookie(accessCookieName)
	if err != nil {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	now := middleware.now().UTC()
	claims, err := middleware.accessTokens.Verify(token, now)
	if err != nil {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	user, err := middleware.users.FindActiveSessionUser(httpContext.Request.Context(), claims.SessionID, now)
	if err != nil && !errors.Is(err, ErrInvalidSession) {
		slog.Error("authentication_session_lookup_failed", "request_id", httpserver.RequestID(httpContext), "error", err)
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return Principal{}, false
	}
	if err != nil || !userCanAuthenticate(user) || user.ID != claims.Subject || user.EstablishmentID != claims.EstablishmentID ||
		user.Role != claims.Role || user.CredentialVersion != claims.CredentialVersion {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return Principal{}, false
	}
	return Principal{
		UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email,
		Role: user.Role, PasswordChangeRequired: user.PasswordChangeRequired,
	}, true
}

func PrincipalFromContext(httpContext *gin.Context) (Principal, bool) {
	value, exists := httpContext.Get(principalContextKey)
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
	return matchesPathRules(csrfProtectedPaths, method, path)
}

func requiresAccessToken(method, path string) bool {
	if isPublicPath(method, path) {
		return false
	}
	return strings.HasPrefix(path, "/api/v1/")
}

func isPublicPath(method, path string) bool {
	return matchesPathRules(publicPaths, method, path)
}

func matchesPathRules(rules []pathRule, method, path string) bool {
	for _, rule := range rules {
		if rule.method != "" && rule.method != method {
			continue
		}
		if rule.path == path || rule.pattern != nil && rule.pattern.MatchString(path) {
			return true
		}
	}
	return false
}
