package health

import (
	"context"
	"net/http"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

type Database interface {
	Ping(context.Context) error
}

type Handler struct {
	database Database
}

func NewHandler(database Database) *Handler {
	return &Handler{database: database}
}

func (h *Handler) GetHealth(c *gin.Context) {
	if err := h.database.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, api.ErrorResponse{
			Code:      "service_unavailable",
			Message:   "Serviço temporariamente indisponível.",
			RequestId: RequestID(c),
		})
		return
	}

	c.JSON(http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}
