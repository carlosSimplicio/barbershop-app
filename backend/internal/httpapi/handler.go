package httpapi

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/establishment"
	"barbershop-app/backend/internal/health"
	"barbershop-app/backend/internal/onboarding"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	health         *health.Handler
	authentication *auth.Handler
	onboarding     *onboarding.Handler
	establishment  *establishment.Handler
}

func NewHandler(healthHandler *health.Handler, authenticationHandler *auth.Handler, onboardingHandler *onboarding.Handler, establishmentHandler *establishment.Handler) *Handler {
	return &Handler{health: healthHandler, authentication: authenticationHandler, onboarding: onboardingHandler, establishment: establishmentHandler}
}

func (handler *Handler) GetHealth(c *gin.Context) {
	handler.health.GetHealth(c)
}

func (handler *Handler) PostAuthLogin(c *gin.Context) {
	handler.authentication.PostAuthLogin(c)
}

func (handler *Handler) PostAuthRegister(c *gin.Context) {
	handler.onboarding.PostAuthRegister(c)
}

func (handler *Handler) PostAuthEmailConfirmation(c *gin.Context) {
	handler.onboarding.PostAuthEmailConfirmation(c)
}

func (handler *Handler) PostAuthEmailConfirmationVerify(c *gin.Context) {
	handler.onboarding.PostAuthEmailConfirmationVerify(c)
}

func (handler *Handler) PostAuthLogout(c *gin.Context) {
	handler.authentication.PostAuthLogout(c)
}

func (handler *Handler) GetAuthMe(c *gin.Context) {
	handler.authentication.GetAuthMe(c)
}

func (handler *Handler) PostAuthPasswordChange(c *gin.Context) {
	handler.authentication.PostAuthPasswordChange(c)
}

func (handler *Handler) GetEstablishment(c *gin.Context) {
	handler.establishment.GetEstablishment(c)
}
