package professionals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/pagination"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
)

type Registry interface {
	Create(context.Context, auth.Principal, CreateInput) (Professional, error)
	List(context.Context, auth.Principal, int32, int32) (ListResult, error)
}

type Handler struct {
	registry Registry
}

type createRequest struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	TemporaryPassword string `json:"temporary_password"`
}

func NewHandler(registry Registry) *Handler {
	return &Handler{registry: registry}
}

func (handler *Handler) GetProfessionals(c *gin.Context, params api.GetProfessionalsParams) {
	principal, ok := auth.RequirePrincipal(c)
	if !ok {
		return
	}
	limit, offset := int32(pagination.DefaultLimit), int32(pagination.DefaultOffset)
	if params.Limit != nil {
		if *params.Limit < pagination.MinLimit || *params.Limit > pagination.MaxLimit {
			writeFieldError(c, http.StatusBadRequest, "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit))
			return
		}
		limit = int32(*params.Limit)
	}
	if params.Offset != nil {
		if *params.Offset < pagination.MinOffset || int64(*params.Offset) > pagination.MaxOffset {
			writeFieldError(c, http.StatusBadRequest, "offset", "invalid_offset", "Informe um deslocamento válido.")
			return
		}
		offset = int32(*params.Offset)
	}

	result, err := handler.registry.List(c.Request.Context(), principal, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeError(c, http.StatusForbidden, "forbidden", "Você não pode consultar profissionais.")
		case errors.Is(err, ErrInvalidLimit):
			writeFieldError(c, http.StatusBadRequest, "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit))
		case errors.Is(err, ErrInvalidOffset):
			writeFieldError(c, http.StatusBadRequest, "offset", "invalid_offset", "Informe um deslocamento válido.")
		default:
			slog.Error("professionals_query_failed", "request_id", httpserver.RequestID(c))
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar os profissionais.")
		}
		return
	}
	items := make([]api.ProfessionalResponse, 0, len(result.Items))
	for _, professional := range result.Items {
		id, err := uuid.Parse(professional.ID)
		if err != nil {
			slog.Error("professional_list_response_invalid", "request_id", httpserver.RequestID(c))
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar os profissionais.")
			return
		}
		items = append(items, api.ProfessionalResponse{
			Id: id, Name: professional.Name, Email: openapiTypes.Email(professional.Email), Active: professional.Active,
		})
	}
	c.JSON(http.StatusOK, api.ProfessionalListResponse{
		Items: items, Limit: int(result.Limit), Offset: int(result.Offset), HasMore: result.HasMore,
	})
}

func (handler *Handler) PostProfessionals(c *gin.Context) {
	principal, ok := auth.RequirePrincipal(c)
	if !ok {
		return
	}
	var request createRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	professional, err := handler.registry.Create(c.Request.Context(), principal, CreateInput{
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
		writeFieldError(c, http.StatusBadRequest, "temporary_password", "invalid_password", "A senha deve ter de 8 a 32 caracteres, sem espaços ou caracteres invisíveis.")
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
