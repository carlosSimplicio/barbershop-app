package auth

import (
	"net/http"
	"time"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
)

const (
	accessCookieName = "__Secure-barbershop_access"
	csrfCookieName   = "__Secure-barbershop_csrf"
)

func WriteSessionResponse(httpContext *gin.Context, result LoginResult) error {
	response, err := principalResponse(result.Principal)
	if err != nil {
		return err
	}
	setSessionCookies(httpContext, result.AccessToken, result.CSRFToken)
	httpContext.JSON(http.StatusOK, response)
	return nil
}

func setSessionCookies(httpContext *gin.Context, accessToken, csrfToken string) {
	http.SetCookie(httpContext.Writer, &http.Cookie{Name: accessCookieName, Value: accessToken, Path: "/api/v1", MaxAge: int(SessionLifetime / time.Second), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(httpContext.Writer, &http.Cookie{Name: csrfCookieName, Value: csrfToken, Path: "/", MaxAge: int(SessionLifetime / time.Second), Secure: true, SameSite: http.SameSiteStrictMode})
}

func clearSessionCookies(httpContext *gin.Context) {
	expires := time.Unix(1, 0).UTC()
	http.SetCookie(httpContext.Writer, &http.Cookie{Name: accessCookieName, Path: "/api/v1", MaxAge: -1, Expires: expires, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(httpContext.Writer, &http.Cookie{Name: csrfCookieName, Path: "/", MaxAge: -1, Expires: expires, Secure: true, SameSite: http.SameSiteStrictMode})
}

func principalResponse(principal Principal) (api.AuthPrincipal, error) {
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return api.AuthPrincipal{}, err
	}
	establishmentID, err := uuid.Parse(principal.EstablishmentID)
	if err != nil {
		return api.AuthPrincipal{}, err
	}
	return api.AuthPrincipal{
		UserId: userID, EstablishmentId: establishmentID, Email: openapiTypes.Email(principal.Email),
		Role: api.UserRole(principal.Role), PasswordChangeRequired: principal.PasswordChangeRequired,
	}, nil
}
