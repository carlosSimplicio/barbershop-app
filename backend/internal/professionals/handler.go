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

func (handler *Handler) GetProfessionals(httpContext *gin.Context, parameters api.GetProfessionalsParams) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	limit, offset := int32(pagination.DefaultLimit), int32(pagination.DefaultOffset)
	if parameters.Limit != nil {
		if *parameters.Limit < pagination.MinLimit || *parameters.Limit > pagination.MaxLimit {
			writeFieldError(httpContext, http.StatusBadRequest, "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit))
			return
		}
		limit = int32(*parameters.Limit)
	}
	if parameters.Offset != nil {
		if *parameters.Offset < pagination.MinOffset || int64(*parameters.Offset) > pagination.MaxOffset {
			writeFieldError(httpContext, http.StatusBadRequest, "offset", "invalid_offset", "Informe um deslocamento válido.")
			return
		}
		offset = int32(*parameters.Offset)
	}

	result, err := handler.registry.List(httpContext.Request.Context(), principal, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeError(httpContext, http.StatusForbidden, "forbidden", "Você não pode consultar profissionais.")
		case errors.Is(err, ErrInvalidLimit):
			writeFieldError(httpContext, http.StatusBadRequest, "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit))
		case errors.Is(err, ErrInvalidOffset):
			writeFieldError(httpContext, http.StatusBadRequest, "offset", "invalid_offset", "Informe um deslocamento válido.")
		default:
			slog.Error("professionals_query_failed", "request_id", httpserver.RequestID(httpContext))
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível consultar os profissionais.")
		}
		return
	}
	items := make([]api.ProfessionalResponse, 0, len(result.Items))
	for _, professional := range result.Items {
		professionalUUID, err := uuid.Parse(professional.ID)
		if err != nil {
			slog.Error("professional_list_response_invalid", "request_id", httpserver.RequestID(httpContext))
			writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível consultar os profissionais.")
			return
		}
		items = append(items, api.ProfessionalResponse{
			Id: professionalUUID, Name: professional.Name, Email: openapiTypes.Email(professional.Email), Active: professional.Active,
		})
	}
	httpContext.JSON(http.StatusOK, api.ProfessionalListResponse{
		Items: items, Limit: int(result.Limit), Offset: int(result.Offset), HasMore: result.HasMore,
	})
}

func (handler *Handler) PostProfessionals(httpContext *gin.Context) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	var request createRequest
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	professional, err := handler.registry.Create(httpContext.Request.Context(), principal, CreateInput{
		Name: request.Name, Email: request.Email, TemporaryPassword: request.TemporaryPassword,
	})
	if err != nil {
		writeCreationError(httpContext, err)
		return
	}
	professionalUUID, err := uuid.Parse(professional.ID)
	if err != nil {
		slog.Error("professional_response_invalid", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir o cadastro.")
		return
	}
	httpContext.JSON(http.StatusCreated, api.ProfessionalResponse{
		Id: professionalUUID, Name: professional.Name, Email: openapiTypes.Email(professional.Email), Active: professional.Active,
	})
}

func writeCreationError(httpContext *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(httpContext, http.StatusForbidden, "forbidden", "Você não pode cadastrar profissionais.")
	case errors.Is(err, entitlements.ErrReadOnly):
		writeError(httpContext, http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura.")
	case errors.Is(err, ErrInvalidName):
		writeFieldError(httpContext, http.StatusBadRequest, "name", "invalid_name", "Informe um nome com até 120 caracteres.")
	case errors.Is(err, auth.ErrInvalidEmail):
		writeFieldError(httpContext, http.StatusBadRequest, "email", "invalid_email", "Informe um e-mail válido.")
	case errors.Is(err, security.ErrInvalidPassword):
		writeFieldError(httpContext, http.StatusBadRequest, "temporary_password", "invalid_password", "A senha deve ter de 8 a 32 caracteres, sem espaços ou caracteres invisíveis.")
	case errors.Is(err, ErrEmailUnavailable):
		writeFieldError(httpContext, http.StatusConflict, "email", "email_unavailable", "Este e-mail já está cadastrado.")
	default:
		slog.Error("professional_registration_failed", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir o cadastro.")
	}
}

func writeError(httpContext *gin.Context, status int, code, message string) {
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}

func writeFieldError(httpContext *gin.Context, status int, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	topCode := "validation_failed"
	if status == http.StatusConflict {
		topCode = "conflict"
	}
	httpContext.JSON(status, api.ErrorResponse{
		Code: topCode, Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(httpContext), Fields: &fields,
	})
}
