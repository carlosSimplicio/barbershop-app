package httpapi

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/health"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	health         *health.Handler
	authentication *auth.Handler
}

func NewHandler(healthHandler *health.Handler, authenticationHandler *auth.Handler) *Handler {
	return &Handler{health: healthHandler, authentication: authenticationHandler}
}

func (handler *Handler) GetHealth(c *gin.Context) {
	handler.health.GetHealth(c)
}

func (handler *Handler) PostAuthLogin(c *gin.Context) {
	handler.authentication.PostAuthLogin(c)
}
