package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/ratelimit"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

type RegistrationRegistrar interface {
	Register(context.Context, RegistrationInput) (RegistrationResult, error)
}

type Handler struct {
	registrations RegistrationRegistrar
}

func NewHandler(registrations RegistrationRegistrar) *Handler {
	return &Handler{registrations: registrations}
}

func (handler *Handler) PostAuthRegister(c *gin.Context) {
	var request api.RegisterRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeRegistrationError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}

	result, err := handler.registrations.Register(c.Request.Context(), RegistrationInput{
		EstablishmentName: request.EstablishmentName, Slug: request.Slug, TimeZone: request.TimeZone,
		Email: string(request.Email), Password: request.Password, IP: c.ClientIP(),
	})
	if err != nil {
		handler.writeRegistrationFailure(c, err)
		return
	}
	if result.DeliveryFailed {
		slog.Warn("registration_confirmation_delivery_failed", "request_id", httpserver.RequestID(c))
	}
	c.Status(http.StatusAccepted)
	c.Writer.WriteHeaderNow()
}

func (handler *Handler) writeRegistrationFailure(c *gin.Context, err error) {
	var rateLimitError ratelimit.Error
	switch {
	case errors.Is(err, ErrInvalidName):
		writeRegistrationFieldError(c, http.StatusBadRequest, "establishment_name", "invalid_name", "Informe um nome com até 120 caracteres.")
	case errors.Is(err, ErrInvalidSlug):
		writeRegistrationFieldError(c, http.StatusBadRequest, "slug", "invalid_slug", "Informe um identificador válido com 3 a 63 caracteres.")
	case errors.Is(err, ErrReservedSlug):
		writeRegistrationFieldError(c, http.StatusBadRequest, "slug", "reserved_slug", "Escolha outro identificador.")
	case errors.Is(err, ErrInvalidTimeZone):
		writeRegistrationFieldError(c, http.StatusBadRequest, "time_zone", "invalid_time_zone", "Informe um fuso horário válido.")
	case errors.Is(err, auth.ErrInvalidEmail):
		writeRegistrationFieldError(c, http.StatusBadRequest, "email", "invalid_email", "Informe um e-mail válido.")
	case errors.Is(err, security.ErrInvalidPassword):
		writeRegistrationFieldError(c, http.StatusBadRequest, "password", "invalid_password", "A senha deve ter de 8 a 32 caracteres e não pode conter espaços.")
	case errors.Is(err, ErrEmailUnavailable):
		writeRegistrationFieldError(c, http.StatusConflict, "email", "email_unavailable", "Este e-mail já está cadastrado.")
	case errors.Is(err, ErrSlugUnavailable):
		writeRegistrationFieldError(c, http.StatusConflict, "slug", "slug_unavailable", "Este identificador não está disponível.")
	case errors.As(err, &rateLimitError):
		retryAfter := int(math.Ceil(rateLimitError.RetryAfter.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("Retry-After", strconv.Itoa(retryAfter))
		writeRegistrationError(c, http.StatusTooManyRequests, "registration_rate_limited", "Muitas tentativas. Tente novamente mais tarde.")
	default:
		slog.Error("registration_failed", "request_id", httpserver.RequestID(c))
		writeRegistrationError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação.")
	}
}

func writeRegistrationError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}

func writeRegistrationFieldError(c *gin.Context, status int, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	topLevelCode := "validation_failed"
	if status == http.StatusConflict {
		topLevelCode = "conflict"
	}
	c.JSON(status, api.ErrorResponse{
		Code: topLevelCode, Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(c), Fields: &fields,
	})
}
