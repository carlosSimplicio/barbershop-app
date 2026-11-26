package offerings

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/pagination"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (handler *Handler) GetProfessionalServices(httpContext *gin.Context, professionalID uuid.UUID, parameters api.GetProfessionalServicesParams) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	limit, offset := int64(pagination.DefaultLimit), int64(pagination.DefaultOffset)
	if parameters.Limit != nil {
		limit = int64(*parameters.Limit)
	}
	if parameters.Offset != nil {
		offset = int64(*parameters.Offset)
	}
	result, err := handler.service.List(httpContext.Request.Context(), principal, professionalID.String(), limit, offset)
	if err != nil {
		writeError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, result)
}
func writeError(httpContext *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação."
	switch {
	case errors.Is(err, ErrForbidden):
		status, code, message = http.StatusForbidden, "forbidden", "Você não pode administrar os serviços deste profissional."
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "Profissional ou serviço não encontrado."
	case errors.Is(err, entitlements.ErrReadOnly):
		status, code, message = http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura."
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidOffset):
		field, fieldCode, fieldMessage := "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit)
		if errors.Is(err, ErrInvalidOffset) {
			field, fieldCode, fieldMessage = "offset", "invalid_offset", "Informe um deslocamento válido."
		}
		fields := []api.FieldError{{Field: field, Code: fieldCode, Message: fieldMessage}}
		httpContext.JSON(http.StatusBadRequest, api.ErrorResponse{Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(httpContext), Fields: &fields})
		return
	default:
		slog.Error("professional_services_operation_failed", "request_id", httpserver.RequestID(httpContext))
	}
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}
