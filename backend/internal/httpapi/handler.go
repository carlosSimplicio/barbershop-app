package httpapi

import (
	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/catalog"
	"barbershop-app/backend/internal/establishment"
	"barbershop-app/backend/internal/health"
	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/professionals"
	api "barbershop-app/backend/openapi/generated"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	catalog        *catalog.Handler
	health         *health.Handler
	authentication *auth.Handler
	onboarding     *onboarding.Handler
	establishment  *establishment.Handler
	professionals  *professionals.Handler
}

func NewHandler(healthHandler *health.Handler, authenticationHandler *auth.Handler, onboardingHandler *onboarding.Handler, establishmentHandler *establishment.Handler, professionalsHandler *professionals.Handler, catalogHandler *catalog.Handler) *Handler {
	return &Handler{catalog: catalogHandler, health: healthHandler, authentication: authenticationHandler, onboarding: onboardingHandler, establishment: establishmentHandler, professionals: professionalsHandler}
}

func (handler *Handler) GetHealth(httpContext *gin.Context) {
	handler.health.GetHealth(httpContext)
}

func (handler *Handler) PostAuthLogin(httpContext *gin.Context) {
	handler.authentication.PostAuthLogin(httpContext)
}

func (handler *Handler) PostAuthPasswordRecovery(httpContext *gin.Context) {
	handler.authentication.PostAuthPasswordRecovery(httpContext)
}

func (handler *Handler) PostAuthPasswordRecoveryVerify(httpContext *gin.Context) {
	handler.authentication.PostAuthPasswordRecoveryVerify(httpContext)
}

func (handler *Handler) PostAuthRegister(httpContext *gin.Context) {
	handler.onboarding.PostAuthRegister(httpContext)
}

func (handler *Handler) PostAuthEmailConfirmation(httpContext *gin.Context) {
	handler.onboarding.PostAuthEmailConfirmation(httpContext)
}

func (handler *Handler) PostAuthEmailConfirmationVerify(httpContext *gin.Context) {
	handler.onboarding.PostAuthEmailConfirmationVerify(httpContext)
}

func (handler *Handler) PostAuthLogout(httpContext *gin.Context) {
	handler.authentication.PostAuthLogout(httpContext)
}

func (handler *Handler) GetAuthMe(httpContext *gin.Context) {
	handler.authentication.GetAuthMe(httpContext)
}

func (handler *Handler) PostAuthPasswordChange(httpContext *gin.Context) {
	handler.authentication.PostAuthPasswordChange(httpContext)
}

func (handler *Handler) GetEstablishment(httpContext *gin.Context) {
	handler.establishment.GetEstablishment(httpContext)
}

func (handler *Handler) GetEstablishmentSettings(httpContext *gin.Context) {
	handler.establishment.GetEstablishmentSettings(httpContext)
}

func (handler *Handler) PatchEstablishmentSettings(httpContext *gin.Context) {
	handler.establishment.PatchEstablishmentSettings(httpContext)
}

func (handler *Handler) PutEstablishmentLogo(httpContext *gin.Context) {
	handler.establishment.PutEstablishmentLogo(httpContext)
}

func (handler *Handler) GetPublicEstablishmentLogo(httpContext *gin.Context, slug string, _ api.GetPublicEstablishmentLogoParams) {
	handler.establishment.GetPublicEstablishmentLogo(httpContext, slug)
}

func (handler *Handler) GetPublicEstablishment(httpContext *gin.Context, slug string) {
	handler.establishment.GetPublicEstablishment(httpContext, slug)
}

func (handler *Handler) PostProfessionals(httpContext *gin.Context) {
	handler.professionals.PostProfessionals(httpContext)
}

func (handler *Handler) GetProfessionals(httpContext *gin.Context, parameters api.GetProfessionalsParams) {
	handler.professionals.GetProfessionals(httpContext, parameters)
}

func (handler *Handler) PostServices(httpContext *gin.Context) {
	handler.catalog.PostServices(httpContext)
}

func (handler *Handler) GetServices(httpContext *gin.Context, parameters api.GetServicesParams) {
	handler.catalog.GetServices(httpContext, parameters)
}

func (handler *Handler) GetService(httpContext *gin.Context, offeringID uuid.UUID) {
	handler.catalog.GetService(httpContext, offeringID)
}

func (handler *Handler) PatchService(httpContext *gin.Context, offeringID uuid.UUID) {
	handler.catalog.PatchService(httpContext, offeringID)
}

func (handler *Handler) PutServiceStatus(httpContext *gin.Context, offeringID uuid.UUID) {
	handler.catalog.PutServiceStatus(httpContext, offeringID)
}

func (handler *Handler) GetProfessional(httpContext *gin.Context, professionalID uuid.UUID) {
	handler.professionals.GetProfessional(httpContext, professionalID)
}

func (handler *Handler) PatchProfessional(httpContext *gin.Context, professionalID uuid.UUID) {
	handler.professionals.PatchProfessional(httpContext, professionalID)
}
