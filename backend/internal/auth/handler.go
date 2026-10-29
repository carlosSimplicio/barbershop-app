package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/ratelimit"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	login                *LoginService
	sessions             *SessionService
	recovery             PasswordRecoveryRequester
	recoveryVerification PasswordRecoveryVerifier
}

type PasswordRecoveryRequester interface {
	Request(ctx context.Context, email, ip string) (PasswordRecoveryRequestResult, error)
}

type PasswordRecoveryVerifier interface {
	Verify(context.Context, string, string) error
}

func NewHandler(login *LoginService, sessions *SessionService, recovery PasswordRecoveryRequester, recoveryVerification PasswordRecoveryVerifier) *Handler {
	return &Handler{login: login, sessions: sessions, recovery: recovery, recoveryVerification: recoveryVerification}
}

func (handler *Handler) PostAuthPasswordRecoveryVerify(httpContext *gin.Context) {
	var request api.PasswordRecoveryVerifyRequest
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	if err := handler.recoveryVerification.Verify(httpContext.Request.Context(), request.Token, request.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrInvalidPasswordRecovery):
			writeError(httpContext, http.StatusBadRequest, "invalid_password_recovery", "Link inválido ou expirado.")
		case errors.Is(err, security.ErrInvalidPassword):
			writeFieldError(httpContext, "new_password", "invalid_password", PasswordRequirementsMessage())
		case errors.Is(err, ErrPasswordUnchanged):
			writeFieldError(httpContext, "new_password", "password_unchanged", "Escolha uma senha diferente da atual.")
		default:
			slog.Error("password_recovery_verification_failed", "request_id", httpserver.RequestID(httpContext))
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}
	httpContext.Status(http.StatusNoContent)
	httpContext.Writer.WriteHeaderNow()
}

func (handler *Handler) PostAuthPasswordRecovery(httpContext *gin.Context) {
	var request api.PasswordRecoveryRequest
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	result, err := handler.recovery.Request(httpContext.Request.Context(), string(request.Email), httpContext.ClientIP())
	if err != nil {
		var rateLimitError ratelimit.Error
		switch {
		case errors.Is(err, ErrInvalidEmail):
			writeFieldError(httpContext, "email", "invalid_email", "Informe um e-mail válido.")
		case errors.As(err, &rateLimitError):
			retryAfter := int(math.Ceil(rateLimitError.RetryAfter.Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			httpContext.Header("Retry-After", strconv.Itoa(retryAfter))
			writeError(httpContext, http.StatusTooManyRequests, "password_recovery_rate_limited", "Muitas tentativas. Tente novamente mais tarde.")
		default:
			slog.Error("password_recovery_request_failed", "request_id", httpserver.RequestID(httpContext))
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}
	if result.DeliveryFailed {
		slog.Warn("password_recovery_delivery_failed", "request_id", httpserver.RequestID(httpContext))
	}
	httpContext.Status(http.StatusAccepted)
	httpContext.Writer.WriteHeaderNow()
}

func (handler *Handler) PostAuthLogin(httpContext *gin.Context) {
	var request api.LoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	if err := ValidateEmail(string(request.Email)); err != nil {
		writeFieldError(httpContext, "email", "invalid_email", "Informe um e-mail válido.")
		return
	}
	if err := security.ValidatePassword(request.Password); err != nil {
		writeFieldError(httpContext, "password", "invalid_password", PasswordRequirementsMessage())
		return
	}

	result, err := handler.login.Login(httpContext.Request.Context(), LoginInput{Email: string(request.Email), Password: request.Password, IP: httpContext.ClientIP()})
	if err != nil {
		var rateLimitError RateLimitError
		switch {
		case errors.As(err, &rateLimitError):
			retryAfter := int(math.Ceil(rateLimitError.RetryAfter.Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			httpContext.Header("Retry-After", strconv.Itoa(retryAfter))
			writeError(httpContext, http.StatusTooManyRequests, "authentication_rate_limited", "Muitas tentativas. Tente novamente mais tarde.")
		case errors.Is(err, ErrInvalidCredentials):
			writeError(httpContext, http.StatusUnauthorized, "invalid_credentials", "E-mail ou senha inválidos.")
		case errors.Is(err, ErrEmailConfirmationRequired):
			writeError(httpContext, http.StatusForbidden, "email_confirmation_required", "Confirme seu e-mail para entrar.")
		default:
			slog.Error("authentication_login_failed", "request_id", httpserver.RequestID(httpContext), "error", err)
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}

	if err := WriteSessionResponse(httpContext, result); err != nil {
		slog.Error("authentication_principal_invalid", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
	}
}

func (handler *Handler) PostAuthLogout(httpContext *gin.Context) {
	accessToken, _ := httpContext.Cookie(accessCookieName)
	if err := handler.sessions.Logout(httpContext.Request.Context(), accessToken); err != nil {
		slog.Error("authentication_logout_failed", "request_id", httpserver.RequestID(httpContext), "error", err)
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return
	}
	clearSessionCookies(httpContext)
	httpContext.Status(http.StatusNoContent)
}

func (handler *Handler) GetAuthMe(httpContext *gin.Context) {
	principal, ok := RequirePrincipal(httpContext)
	if !ok {
		return
	}
	response, err := principalResponse(principal)
	if err != nil {
		slog.Error("authentication_principal_invalid", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		return
	}
	httpContext.JSON(http.StatusOK, response)
}

func (handler *Handler) PostAuthPasswordChange(httpContext *gin.Context) {
	principal, ok := RequirePrincipal(httpContext)
	if !ok {
		return
	}
	var request api.ChangePasswordRequest
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	result, err := handler.sessions.ChangeTemporaryPassword(httpContext.Request.Context(), principal, request.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, security.ErrInvalidPassword):
			writeFieldError(httpContext, "new_password", "invalid_password", PasswordRequirementsMessage())
		case errors.Is(err, ErrPasswordUnchanged):
			writeFieldError(httpContext, "new_password", "password_unchanged", "Escolha uma senha diferente da senha temporária.")
		case errors.Is(err, ErrPasswordChangeDenied), errors.Is(err, ErrInvalidSession):
			writeError(httpContext, http.StatusForbidden, "password_change_not_allowed", "A troca de senha não está disponível.")
		default:
			slog.Error("authentication_password_change_failed", "request_id", httpserver.RequestID(httpContext), "error", err)
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
		}
		return
	}
	setSessionCookies(httpContext, result.AccessToken, result.CSRFToken)
	httpContext.Status(http.StatusNoContent)
}

func writeError(httpContext *gin.Context, status int, code, message string) {
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}

func writeFieldError(httpContext *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	httpContext.JSON(http.StatusBadRequest, api.ErrorResponse{
		Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(httpContext), Fields: &fields,
	})
}
