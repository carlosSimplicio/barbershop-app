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
	accessCookieName  = "__Secure-barbershop_access"
	refreshCookieName = "__Secure-barbershop_refresh"
	csrfCookieName    = "__Secure-barbershop_csrf"
)

func WriteSessionResponse(c *gin.Context, result LoginResult) error {
	response, err := principalResponse(result.Principal)
	if err != nil {
		return err
	}
	setSessionCookies(c, result.AccessToken, result.RefreshToken, result.CSRFToken)
	c.JSON(http.StatusOK, response)
	return nil
}

func setSessionCookies(c *gin.Context, accessToken, refreshToken, csrfToken string) {
	http.SetCookie(c.Writer, &http.Cookie{Name: accessCookieName, Value: accessToken, Path: "/api/v1", MaxAge: 15 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: refreshToken, Path: "/api/v1/auth", MaxAge: 7 * 24 * 60 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: csrfCookieName, Value: csrfToken, Path: "/", MaxAge: 15 * 24 * 60 * 60, Secure: true, SameSite: http.SameSiteStrictMode})
}

func clearSessionCookies(c *gin.Context) {
	expires := time.Unix(1, 0).UTC()
	http.SetCookie(c.Writer, &http.Cookie{Name: accessCookieName, Path: "/api/v1", MaxAge: -1, Expires: expires, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Path: "/api/v1/auth", MaxAge: -1, Expires: expires, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: csrfCookieName, Path: "/", MaxAge: -1, Expires: expires, Secure: true, SameSite: http.SameSiteStrictMode})
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
