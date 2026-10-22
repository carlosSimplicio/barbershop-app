package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/httpserver"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	login    *LoginService
	sessions *SessionService
}

func NewHandler(login *LoginService, sessions *SessionService) *Handler {
	return &Handler{login: login, sessions: sessions}
}

func (handler *Handler) PostAuthLogin(c *gin.Context) {
	var request api.LoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	if err := ValidateEmail(string(request.Email)); err != nil {
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
			slog.Error("authentication_login_failed", "request_id", httpserver.RequestID(c), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}

	if err := WriteSessionResponse(c, result); err != nil {
		slog.Error("authentication_principal_invalid", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
	}
}

func (handler *Handler) PostAuthLogout(c *gin.Context) {
	accessToken, _ := c.Cookie(accessCookieName)
	if err := handler.sessions.Logout(c.Request.Context(), accessToken); err != nil {
		slog.Error("authentication_logout_failed", "request_id", httpserver.RequestID(c), "error", err)
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return
	}
	clearSessionCookies(c)
	c.Status(http.StatusNoContent)
}

func (handler *Handler) GetAuthMe(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	response, err := principalResponse(principal)
	if err != nil {
		slog.Error("authentication_principal_invalid", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return
	}
	c.JSON(http.StatusOK, response)
}

func (handler *Handler) PostAuthPasswordChange(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request api.ChangePasswordRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	result, err := handler.sessions.ChangeTemporaryPassword(c.Request.Context(), principal, request.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, security.ErrInvalidPassword):
			writeFieldError(c, "new_password", "invalid_password", "A senha deve ter de 8 a 32 caracteres e não pode conter espaços.")
		case errors.Is(err, ErrPasswordUnchanged):
			writeFieldError(c, "new_password", "password_unchanged", "Escolha uma senha diferente da senha temporária.")
		case errors.Is(err, ErrPasswordChangeDenied), errors.Is(err, ErrInvalidSession):
			writeError(c, http.StatusForbidden, "password_change_not_allowed", "A troca de senha não está disponível.")
		default:
			slog.Error("authentication_password_change_failed", "request_id", httpserver.RequestID(c), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}
	setSessionCookies(c, result.AccessToken, result.CSRFToken)
	c.Status(http.StatusNoContent)
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}

func writeFieldError(c *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	c.JSON(http.StatusBadRequest, api.ErrorResponse{
		Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(c), Fields: &fields,
	})
}
