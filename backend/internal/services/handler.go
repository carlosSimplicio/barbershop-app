package services

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/platform/httpserver"
	api "barbershop-app/backend/openapi/generated"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

var ErrInvalidRequest = errors.New("invalid service request")

type createRequest struct {
	Name            string `json:"name"`
	PriceCents      *int64 `json:"price_cents"`
	DurationMinutes *int64 `json:"duration_minutes"`
}

func decodeRequest(c *gin.Context, request any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		var field *json.UnmarshalTypeError
		if errors.As(err, &field) {
			switch field.Field {
			case "name":
				return ErrInvalidName
			case "price_cents":
				return ErrInvalidPrice
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
func (h *Handler) PostServices(c *gin.Context) {
	p, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request createRequest
	if err := decodeRequest(c, &request); err != nil {
		writeServiceError(c, err)
		return
	}
	record, err := h.service.Create(c.Request.Context(), p, CreateInput{Name: request.Name, PriceCents: request.PriceCents, DurationMinutes: request.DurationMinutes})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, record)
}
func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(c, 404, "not_found", "Serviço não encontrado.")
	case errors.Is(err, ErrForbidden):
		writeError(c, 403, "forbidden", "Você não pode administrar serviços.")
	case errors.Is(err, entitlements.ErrReadOnly):
		writeError(c, 403, "read_only", "O estabelecimento está em modo somente leitura.")
	case errors.Is(err, ErrInvalidLimit):
		writeFieldError(c, "limit", "invalid_limit", "Informe um limite entre 1 e 100.")
	case errors.Is(err, ErrInvalidOffset):
		writeFieldError(c, "offset", "invalid_offset", "Informe um deslocamento válido.")
	case errors.Is(err, ErrInvalidName):
		writeFieldError(c, "name", "invalid_name", "Informe um nome com até 120 caracteres.")
	case errors.Is(err, ErrInvalidPrice):
		writeFieldError(c, "price_cents", "invalid_price", "Informe um preço de R$ 0,00 a R$ 99.999,99.")
	case errors.Is(err, ErrInvalidDuration):
		writeFieldError(c, "duration_minutes", "invalid_duration", "Informe de 1 a 1.440 minutos inteiros.")
	case errors.Is(err, ErrInvalidRequest):
		writeError(c, 400, "invalid_request", "Verifique os dados informados.")
	default:
		slog.Error("service_operation_failed", "request_id", httpserver.RequestID(c))
		writeError(c, 500, "internal_error", "Não foi possível concluir a operação.")
	}
}
func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}
func writeFieldError(c *gin.Context, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	c.JSON(400, api.ErrorResponse{Code: "validation_failed", Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(c), Fields: &fields})
}

func (h *Handler) GetServices(c *gin.Context, params api.GetServicesParams) {
	p, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, 401, "authentication_required", "Entre novamente para continuar.")
		return
	}
	limit, offset := int64(20), int64(0)
	if params.Limit != nil {
		limit = int64(*params.Limit)
	}
	if params.Offset != nil {
		offset = int64(*params.Offset)
	}
	page, err := h.service.List(c.Request.Context(), p, limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, page)
}

func (h *Handler) GetService(c *gin.Context, id uuid.UUID) {
	p, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, 401, "authentication_required", "Entre novamente para continuar.")
		return
	}
	record, err := h.service.Get(c.Request.Context(), p, id.String())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, record)
}
