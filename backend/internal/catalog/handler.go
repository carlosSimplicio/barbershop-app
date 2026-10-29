package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

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

type createRequest struct {
	Name            string `json:"name"`
	PriceCents      *int64 `json:"price_cents"`
	DurationMinutes *int64 `json:"duration_minutes"`
}

func decodeRequest(httpContext *gin.Context, request any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &typeError) {
			switch typeError.Field {
			case "name":
				return ErrInvalidName
			case "price_cents":
				return ErrInvalidPrice
			case "active":
				return ErrInvalidStatus
			case "duration_minutes":
				return ErrInvalidDuration
			}
		}
		return ErrInvalidRequest
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return ErrInvalidRequest
	}
	return nil
}

func (handler *Handler) PostServices(httpContext *gin.Context) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	var request createRequest
	if err := decodeRequest(httpContext, &request); err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	record, err := handler.service.Create(httpContext.Request.Context(), principal, CreateInput{Name: request.Name, PriceCents: request.PriceCents, DurationMinutes: request.DurationMinutes})
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusCreated, record)
}

func writeCatalogError(httpContext *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(httpContext, http.StatusNotFound, "not_found", "Serviço não encontrado.")
	case errors.Is(err, ErrForbidden):
		writeError(httpContext, http.StatusForbidden, "forbidden", "Você não pode administrar serviços.")
	case errors.Is(err, entitlements.ErrReadOnly):
		writeError(httpContext, http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura.")
	case errors.Is(err, ErrInvalidLimit):
		writeFieldError(httpContext, "limit", "invalid_limit", fmt.Sprintf("Informe um limite entre %d e %d.", pagination.MinLimit, pagination.MaxLimit))
	case errors.Is(err, ErrInvalidOffset):
		writeFieldError(httpContext, "offset", "invalid_offset", "Informe um deslocamento válido.")
	case errors.Is(err, ErrInvalidName):
		writeFieldError(httpContext, "name", "invalid_name", fmt.Sprintf("Informe um nome com até %d caracteres.", MaxNameCodePoints))
	case errors.Is(err, ErrInvalidPrice):
		writeFieldError(httpContext, "price_cents", "invalid_price", fmt.Sprintf("Informe um preço de %s a %s.", formatBRLBound(MinPriceCents), formatBRLBound(MaxPriceCents)))
	case errors.Is(err, ErrInvalidDuration):
		writeFieldError(httpContext, "duration_minutes", "invalid_duration", fmt.Sprintf("Informe de %s a %s minutos inteiros.", formatIntegerBound(MinDurationMinutes), formatIntegerBound(MaxDurationMinutes)))
	case errors.Is(err, ErrInvalidStatus):
		writeFieldError(httpContext, "active", "invalid_status", "Informe o estado ativo do serviço.")
	case errors.Is(err, ErrInvalidRequest):
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
	default:
		slog.Error("catalog_operation_failed", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}

func formatIntegerBound(value int64) string {
	digits := strconv.FormatInt(value, 10)
	for position := len(digits) - 3; position > 0; position -= 3 {
		digits = digits[:position] + "." + digits[position:]
	}
	return digits
}

func formatBRLBound(cents int64) string {
	return fmt.Sprintf("R$ %s,%02d", formatIntegerBound(cents/100), cents%100)
}

func writeError(httpContext *gin.Context, status int, code, message string) {
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}

func writeFieldError(httpContext *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	httpContext.JSON(http.StatusBadRequest, api.ErrorResponse{Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(httpContext), Fields: &fields})
}

func (handler *Handler) GetServices(httpContext *gin.Context, parameters api.GetServicesParams) {
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
	page, err := handler.service.List(httpContext.Request.Context(), principal, limit, offset)
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, page)
}

func (handler *Handler) GetService(httpContext *gin.Context, offeringID uuid.UUID) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	record, err := handler.service.Get(httpContext.Request.Context(), principal, offeringID.String())
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, record)
}

// RawMessage preserves omitted fields separately from explicit null until each
// field is decoded into the typed update input.
type patchRequest struct {
	Name            json.RawMessage `json:"name"`
	PriceCents      json.RawMessage `json:"price_cents"`
	DurationMinutes json.RawMessage `json:"duration_minutes"`
}

func decodePatch(httpContext *gin.Context) (UpdateInput, error) {
	var requestFields patchRequest
	var input UpdateInput
	if err := decodeRequest(httpContext, &requestFields); err != nil {
		return input, err
	}
	if requestFields.Name != nil {
		if err := json.Unmarshal(requestFields.Name, &input.Name); err != nil || input.Name == nil {
			return input, ErrInvalidName
		}
	}
	if requestFields.PriceCents != nil {
		if err := json.Unmarshal(requestFields.PriceCents, &input.PriceCents); err != nil || input.PriceCents == nil {
			return input, ErrInvalidPrice
		}
	}
	if requestFields.DurationMinutes != nil {
		input.SetDuration = true
		if err := json.Unmarshal(requestFields.DurationMinutes, &input.DurationMinutes); err != nil {
			return input, ErrInvalidDuration
		}
	}
	return input, nil
}

func (handler *Handler) PatchService(httpContext *gin.Context, offeringID uuid.UUID) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	input, err := decodePatch(httpContext)
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	record, err := handler.service.Update(httpContext.Request.Context(), principal, offeringID.String(), input)
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, record)
}

func (handler *Handler) PutServiceStatus(httpContext *gin.Context, offeringID uuid.UUID) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	var request struct {
		Active *bool `json:"active"`
	}
	if err := decodeRequest(httpContext, &request); err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	record, err := handler.service.SetStatus(httpContext.Request.Context(), principal, offeringID.String(), request.Active)
	if err != nil {
		writeCatalogError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, record)
}
