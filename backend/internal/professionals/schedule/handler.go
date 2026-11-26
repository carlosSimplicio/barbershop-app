package schedule

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

const MaxRequestBytes = 640 * 1024

var (
	errInvalidRequest  = errors.New("invalid weekly schedule request")
	errRequestTooLarge = errors.New("weekly schedule request too large")
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) GetProfessionalWeeklySchedule(httpContext *gin.Context, professionalID uuid.UUID) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	week, err := handler.service.Read(httpContext.Request.Context(), principal, professionalID)
	if err != nil {
		writeError(httpContext, err)
		return
	}
	writeWeek(httpContext, professionalID, week)
}

func (handler *Handler) PutProfessionalWeeklySchedule(httpContext *gin.Context, professionalID uuid.UUID) {
	principal, ok := auth.RequirePrincipal(httpContext)
	if !ok {
		return
	}
	input, err := decodeWeek(httpContext)
	if err != nil {
		writeError(httpContext, err)
		return
	}
	week, err := handler.service.Replace(httpContext.Request.Context(), principal, professionalID, input)
	if err != nil {
		writeError(httpContext, err)
		return
	}
	writeWeek(httpContext, professionalID, week)
}

func decodeWeek(httpContext *gin.Context) ([]periodInput, error) {
	if httpContext.Request.ContentLength > MaxRequestBytes {
		return nil, errRequestTooLarge
	}
	body := http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, MaxRequestBytes)
	payload, err := io.ReadAll(body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, errRequestTooLarge
		}
		return nil, errInvalidRequest
	}
	week, err := decodeObject(payload, 1)
	if err != nil {
		return nil, errInvalidRequest
	}
	var rawPeriods []json.RawMessage
	if err := json.Unmarshal(week["periods"], &rawPeriods); err != nil || rawPeriods == nil {
		return nil, errInvalidRequest
	}
	if len(rawPeriods) > maxPeriods {
		return nil, ErrInvalidPeriods
	}
	periods := make([]periodInput, 0, len(rawPeriods))
	for _, rawPeriod := range rawPeriods {
		fields, err := decodeObject(rawPeriod, 3)
		if err != nil {
			return nil, errInvalidRequest
		}
		var weekday *int
		var start, end *string
		if json.Unmarshal(fields["weekday"], &weekday) != nil || weekday == nil ||
			json.Unmarshal(fields["start_time"], &start) != nil || start == nil ||
			json.Unmarshal(fields["end_time"], &end) != nil || end == nil {
			return nil, errInvalidRequest
		}
		periods = append(periods, periodInput{Weekday: *weekday, StartTime: *start, EndTime: *end})
	}
	return periods, nil
}

// Decode keys explicitly to enforce the contract's casing and prevent a
// duplicate key from hiding an invalid or null value earlier in the object.
func decodeObject(payload []byte, fieldCount int) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if opening, err := decoder.Token(); err != nil || opening != json.Delim('{') {
		return nil, errInvalidRequest
	}
	fields := make(map[string]json.RawMessage, fieldCount)
	for decoder.More() {
		token, err := decoder.Token()
		key, isKey := token.(string)
		if err != nil || !isKey || len(fields) == fieldCount {
			return nil, errInvalidRequest
		}
		if _, duplicate := fields[key]; duplicate {
			return nil, errInvalidRequest
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, errInvalidRequest
		}
		fields[key] = value
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') || len(fields) != fieldCount {
		return nil, errInvalidRequest
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errInvalidRequest
	}
	return fields, nil
}

func writeWeek(httpContext *gin.Context, professionalID uuid.UUID, week Week) {
	periods := make([]api.ProfessionalWorkPeriod, 0, len(week.Periods))
	for _, period := range week.Periods {
		periods = append(periods, api.ProfessionalWorkPeriod{
			Weekday:   int(period.Weekday),
			StartTime: formatLocalTime(period.StartMinute),
			EndTime:   formatLocalTime(period.EndMinute),
		})
	}
	httpContext.JSON(http.StatusOK, api.ProfessionalWeeklyScheduleResponse{
		ProfessionalId: professionalID,
		TimeZone:       week.TimeZone,
		CanWrite:       week.CanWrite,
		Periods:        periods,
	})
}

func formatLocalTime(minute int16) string {
	return fmt.Sprintf("%02d:%02d", minute/minutesPerHour, minute%minutesPerHour)
}

func writeError(httpContext *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação."
	switch {
	case errors.Is(err, errInvalidRequest):
		status, code, message = http.StatusBadRequest, "invalid_request", "Informe os períodos da jornada."
	case errors.Is(err, ErrInvalidPeriods):
		fields := []api.FieldError{{Field: "periods", Code: "invalid_periods", Message: "Informe períodos válidos, sem sobreposição no mesmo dia."}}
		httpContext.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code: "validation_failed", Message: "Verifique os períodos da jornada.", RequestId: httpserver.RequestID(httpContext), Fields: &fields,
		})
		return
	case errors.Is(err, errRequestTooLarge):
		status, code, message = http.StatusRequestEntityTooLarge, "request_too_large", "A jornada excede o tamanho permitido."
	case errors.Is(err, ErrStalePrincipal):
		status, code, message = http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar."
	case errors.Is(err, entitlements.ErrReadOnly):
		status, code, message = http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura."
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "Profissional não encontrado."
	default:
		slog.Error("professional_schedule_operation_failed", "request_id", httpserver.RequestID(httpContext))
	}
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}
