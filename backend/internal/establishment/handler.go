package establishment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/platform/httpserver"
	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Finder interface {
	Get(context.Context, auth.Principal) (Establishment, error)
}

type SettingsFinder interface {
	GetSettings(context.Context, auth.Principal) (EstablishmentSettings, error)
}

type Handler struct {
	finder         Finder
	settingsFinder SettingsFinder
}

func NewHandler(finder Finder, settingsFinder SettingsFinder) *Handler {
	return &Handler{finder: finder, settingsFinder: settingsFinder}
}

func (handler *Handler) GetEstablishment(c *gin.Context) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	establishment, err := handler.finder.Get(c.Request.Context(), principal)
	if err != nil {
		slog.Error("establishment_query_failed", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar o estabelecimento.")
		return
	}
	id, err := uuid.Parse(establishment.ID)
	if err != nil {
		slog.Error("establishment_response_invalid", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar o estabelecimento.")
		return
	}

	response := api.EstablishmentResponse{
		Id: id, Name: establishment.Name, Slug: establishment.Slug, TimeZone: establishment.TimeZone,
	}
	if establishment.Trial != nil {
		response.Trial = &api.EstablishmentTrial{
			Status: api.TrialStatus(establishment.Trial.Status), StartsAt: establishment.Trial.StartsAt.UTC(), EndsAt: establishment.Trial.EndsAt.UTC(),
		}
	}
	c.JSON(http.StatusOK, response)
}

func (handler *Handler) GetEstablishmentSettings(c *gin.Context) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	settings, err := handler.settingsFinder.GetSettings(c.Request.Context(), principal)
	if err != nil {
		if errors.Is(err, ErrSettingsForbidden) {
			writeError(c, http.StatusForbidden, "forbidden", "Você não pode consultar as configurações do estabelecimento.")
			return
		}
		slog.Error("establishment_settings_query_failed", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar as configurações do estabelecimento.")
		return
	}
	id, err := uuid.Parse(settings.ID)
	if err != nil {
		slog.Error("establishment_settings_response_invalid", "request_id", httpserver.RequestID(c))
		writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível consultar as configurações do estabelecimento.")
		return
	}
	links := make([]api.EstablishmentSocialLink, 0, len(settings.SocialLinks))
	for _, link := range settings.SocialLinks {
		links = append(links, api.EstablishmentSocialLink{Label: link.Label, Url: link.URL})
	}
	var logo *api.EstablishmentLogoMetadata
	if settings.Logo != nil {
		logo = &api.EstablishmentLogoMetadata{
			Url: settings.Logo.URL, ContentType: api.EstablishmentLogoMetadataContentType(settings.Logo.ContentType),
			ByteSize: settings.Logo.ByteSize, Width: settings.Logo.Width, Height: settings.Logo.Height,
		}
	}
	c.JSON(http.StatusOK, api.EstablishmentSettingsResponse{
		Id: id, Name: settings.Name, Slug: settings.Slug, TimeZone: settings.TimeZone,
		Description: settings.Description, Address: settings.Address, ContactPhone: settings.ContactPhone,
		SocialLinks: links, MinimumBookingNoticeMinutes: settings.MinimumBookingNoticeMinutes,
		BookingHorizonDays: settings.BookingHorizonDays, CustomerChangeDeadlineMinutes: settings.CustomerChangeDeadlineMinutes,
		Logo: logo, TimeZoneEditable: settings.TimeZoneEditable, CanWrite: settings.CanWrite,
		SlugConflict: settings.SlugConflict, IsComplete: settings.IsComplete, IsPublishable: settings.IsPublishable,
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}
