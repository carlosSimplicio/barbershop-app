package services

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/httpserver"
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
	principal, ok := auth.PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request createRequest
	if err := decodeRequest(httpContext, &request); err != nil {
		writeServiceError(httpContext, err)
		return
	}
	record, err := handler.service.Create(httpContext.Request.Context(), principal, CreateInput{Name: request.Name, PriceCents: request.PriceCents, DurationMinutes: request.DurationMinutes})
	if err != nil {
		writeServiceError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusCreated, record)
}

func writeServiceError(httpContext *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(httpContext, http.StatusNotFound, "not_found", "Serviço não encontrado.")
	case errors.Is(err, ErrForbidden):
		writeError(httpContext, http.StatusForbidden, "forbidden", "Você não pode administrar serviços.")
	case errors.Is(err, entitlements.ErrReadOnly):
		writeError(httpContext, http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura.")
	case errors.Is(err, ErrInvalidLimit):
		writeFieldError(httpContext, "limit", "invalid_limit", "Informe um limite entre 1 e 100.")
	case errors.Is(err, ErrInvalidOffset):
		writeFieldError(httpContext, "offset", "invalid_offset", "Informe um deslocamento válido.")
	case errors.Is(err, ErrInvalidName):
		writeFieldError(httpContext, "name", "invalid_name", "Informe um nome com até 120 caracteres.")
	case errors.Is(err, ErrInvalidPrice):
		writeFieldError(httpContext, "price_cents", "invalid_price", "Informe um preço de R$ 0,00 a R$ 99.999,99.")
	case errors.Is(err, ErrInvalidDuration):
		writeFieldError(httpContext, "duration_minutes", "invalid_duration", "Informe de 1 a 1.440 minutos inteiros.")
	case errors.Is(err, ErrInvalidStatus):
		writeFieldError(httpContext, "active", "invalid_status", "Informe o estado ativo do serviço.")
	case errors.Is(err, ErrInvalidRequest):
		writeError(httpContext, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
	default:
		slog.Error("service_operation_failed", "request_id", httpserver.RequestID(httpContext))
		writeError(httpContext, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}

func writeError(httpContext *gin.Context, status int, code, message string) {
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}

func writeFieldError(httpContext *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	httpContext.JSON(http.StatusBadRequest, api.ErrorResponse{Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(httpContext), Fields: &fields})
}

func (handler *Handler) GetServices(httpContext *gin.Context, parameters api.GetServicesParams) {
	principal, ok := auth.PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	limit, offset := int64(20), int64(0)
	if parameters.Limit != nil {
		limit = int64(*parameters.Limit)
	}
	if parameters.Offset != nil {
		offset = int64(*parameters.Offset)
	}
	page, err := handler.service.List(httpContext.Request.Context(), principal, limit, offset)
	if err != nil {
		writeServiceError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, page)
}

func (handler *Handler) GetService(httpContext *gin.Context, offeringID uuid.UUID) {
	principal, ok := auth.PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	record, err := handler.service.Get(httpContext.Request.Context(), principal, offeringID.String())
	if err != nil {
		writeServiceError(httpContext, err)
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
	principal, ok := auth.PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	input, err := decodePatch(httpContext)
	if err != nil {
		writeServiceError(httpContext, err)
		return
	}
	record, err := handler.service.Update(httpContext.Request.Context(), principal, offeringID.String(), input)
	if err != nil {
		writeServiceError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, record)
}

func (handler *Handler) PutServiceStatus(httpContext *gin.Context, offeringID uuid.UUID) {
	principal, ok := auth.PrincipalFromContext(httpContext)
	if !ok {
		writeError(httpContext, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request struct {
		Active *bool `json:"active"`
	}
	if err := decodeRequest(httpContext, &request); err != nil {
		writeServiceError(httpContext, err)
		return
	}
	record, err := handler.service.SetStatus(httpContext.Request.Context(), principal, offeringID.String(), request.Active)
	if err != nil {
		writeServiceError(httpContext, err)
		return
	}
	httpContext.JSON(http.StatusOK, record)
}
