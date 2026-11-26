package schedule

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/platform/httpserver"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const minutesPerHour = 60

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
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "Profissional não encontrado."
	default:
		slog.Error("professional_schedule_operation_failed", "request_id", httpserver.RequestID(httpContext))
	}
	httpContext.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(httpContext)})
}
