package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"barbershop-app/backend/internal/auth/security"
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

type Handler struct {
	login *LoginService
}

func NewHandler(login *LoginService) *Handler {
	return &Handler{login: login}
}

func (handler *Handler) PostAuthLogin(c *gin.Context) {
	var request api.LoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	rawEmail := strings.TrimSpace(string(request.Email))
	if parsed, err := mail.ParseAddress(rawEmail); err != nil || parsed.Address != rawEmail || len(rawEmail) > 254 {
		writeFieldError(c, "email", "invalid_email", "Informe um e-mail válido.")
		return
	}
	if err := security.ValidatePassword(request.Password); err != nil {
		writeFieldError(c, "password", "invalid_password", "A senha deve ter de 8 a 32 caracteres e não pode conter espaços.")
		return
	}

	result, err := handler.login.Login(c.Request.Context(), LoginInput{Email: string(request.Email), Password: request.Password, IP: c.ClientIP()})
	if err != nil {
		var rateLimitError RateLimitError
		switch {
		case errors.As(err, &rateLimitError):
			retryAfter := int(math.Ceil(rateLimitError.RetryAfter.Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			writeError(c, http.StatusTooManyRequests, "authentication_rate_limited", "Muitas tentativas. Tente novamente mais tarde.")
		case errors.Is(err, ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "invalid_credentials", "E-mail ou senha inválidos.")
		case errors.Is(err, ErrEmailConfirmationRequired):
			writeError(c, http.StatusForbidden, "email_confirmation_required", "Confirme seu e-mail para entrar.")
		default:
			slog.Error("authentication_login_failed", "request_id", requestID(c), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}

	userID, userIDError := uuid.Parse(result.Principal.UserID)
	establishmentID, establishmentIDError := uuid.Parse(result.Principal.EstablishmentID)
	if userIDError != nil || establishmentIDError != nil {
		slog.Error("authentication_principal_invalid", "request_id", requestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return
	}
	setSessionCookies(c, result.AccessToken, result.RefreshToken, result.CSRFToken)
	c.JSON(http.StatusOK, api.AuthPrincipal{
		UserId: userID, EstablishmentId: establishmentID,
		Email: openapiTypes.Email(result.Principal.Email), Role: api.UserRole(result.Principal.Role),
		PasswordChangeRequired: result.Principal.PasswordChangeRequired,
	})
}

func setSessionCookies(c *gin.Context, accessToken, refreshToken, csrfToken string) {
	http.SetCookie(c.Writer, &http.Cookie{Name: accessCookieName, Value: accessToken, Path: "/api/v1", MaxAge: 15 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: refreshToken, Path: "/api/v1/auth", MaxAge: 7 * 24 * 60 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: csrfCookieName, Value: csrfToken, Path: "/", MaxAge: 7 * 24 * 60 * 60, Secure: true, SameSite: http.SameSiteStrictMode})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: requestID(c)})
}

func writeFieldError(c *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	c.JSON(http.StatusBadRequest, api.ErrorResponse{
		Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: requestID(c), Fields: &fields,
	})
}

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}
