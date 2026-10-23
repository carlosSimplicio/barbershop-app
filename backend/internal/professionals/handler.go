package professionals

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/httpserver"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
)

type Creator interface {
	Create(context.Context, auth.Principal, CreateInput) (Professional, error)
}

type Handler struct {
	creator Creator
}

type createRequest struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	TemporaryPassword string `json:"temporary_password"`
}

func NewHandler(creator Creator) *Handler {
	return &Handler{creator: creator}
}

func (handler *Handler) PostProfessionals(c *gin.Context) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request createRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	professional, err := handler.creator.Create(c.Request.Context(), principal, CreateInput{
		Name: request.Name, Email: request.Email, TemporaryPassword: request.TemporaryPassword,
	})
	if err != nil {
		writeCreationError(c, err)
		return
	}
	id, err := uuid.Parse(professional.ID)
	if err != nil {
		slog.Error("professional_response_invalid", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir o cadastro.")
		return
	}
	c.JSON(http.StatusCreated, api.ProfessionalResponse{
		Id: id, Name: professional.Name, Email: openapiTypes.Email(professional.Email), Active: professional.Active,
	})
}

func writeCreationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", "Você não pode cadastrar profissionais.")
	case errors.Is(err, entitlements.ErrReadOnly):
		writeError(c, http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura.")
	case errors.Is(err, ErrInvalidName):
		writeFieldError(c, http.StatusBadRequest, "name", "invalid_name", "Informe um nome com até 120 caracteres.")
	case errors.Is(err, auth.ErrInvalidEmail):
		writeFieldError(c, http.StatusBadRequest, "email", "invalid_email", "Informe um e-mail válido.")
	case errors.Is(err, security.ErrInvalidPassword):
		writeFieldError(c, http.StatusBadRequest, "temporary_password", "invalid_password", "A senha deve ter de 8 a 32 caracteres e não pode conter espaços.")
	case errors.Is(err, ErrEmailUnavailable):
		writeFieldError(c, http.StatusConflict, "email", "email_unavailable", "Este e-mail já está cadastrado.")
	default:
		slog.Error("professional_registration_failed", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível concluir o cadastro.")
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}

func writeFieldError(c *gin.Context, status int, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	topCode := "validation_failed"
	if status == http.StatusConflict {
		topCode = "conflict"
	}
	c.JSON(status, api.ErrorResponse{
		Code: topCode, Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(c), Fields: &fields,
	})
}
