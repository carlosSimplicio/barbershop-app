package establishment

import (
	"bytes"
	"context"
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

type Finder interface {
	Get(context.Context, auth.Principal) (Establishment, error)
}

type SettingsFinder interface {
	GetSettings(context.Context, auth.Principal) (EstablishmentSettings, error)
}

type SettingsUpdater interface {
	PatchSettings(context.Context, auth.Principal, SettingsPatch) error
}

type SettingsManager interface {
	SettingsFinder
	SettingsUpdater
}

type Handler struct {
	finder   Finder
	settings SettingsManager
}

func NewHandler(finder Finder, settings SettingsManager) *Handler {
	return &Handler{finder: finder, settings: settings}
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
	settings, err := handler.settings.GetSettings(c.Request.Context(), principal)
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

type settingsPatchRequest struct {
	Name                          json.RawMessage `json:"name"`
	TimeZone                      json.RawMessage `json:"time_zone"`
	Description                   json.RawMessage `json:"description"`
	Address                       json.RawMessage `json:"address"`
	ContactPhone                  json.RawMessage `json:"contact_phone"`
	SocialLinks                   json.RawMessage `json:"social_links"`
	MinimumBookingNoticeMinutes   json.RawMessage `json:"minimum_booking_notice_minutes"`
	BookingHorizonDays            json.RawMessage `json:"booking_horizon_days"`
	CustomerChangeDeadlineMinutes json.RawMessage `json:"customer_change_deadline_minutes"`
}

func (handler *Handler) PatchEstablishmentSettings(c *gin.Context) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication_required", "Entre novamente para continuar.")
		return
	}
	var request settingsPatchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 65536))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(c, http.StatusBadRequest, "invalid_request", "Verifique os dados informados.")
		return
	}
	patch, fieldError := decodeSettingsPatch(request)
	if fieldError != nil {
		writeFieldError(c, http.StatusBadRequest, fieldError.Field, fieldError.Code, fieldError.Message)
		return
	}
	if err := handler.settings.PatchSettings(c.Request.Context(), principal, patch); err != nil {
		switch {
		case errors.Is(err, ErrSettingsForbidden):
			writeError(c, http.StatusForbidden, "forbidden", "Você não pode alterar as configurações do estabelecimento.")
		case errors.Is(err, entitlements.ErrReadOnly):
			writeError(c, http.StatusForbidden, "read_only", "O estabelecimento está em modo somente leitura.")
		case errors.Is(err, ErrTimeZoneLocked):
			writeFieldError(c, http.StatusConflict, "time_zone", "time_zone_locked", "O fuso não pode ser alterado após o primeiro dado operacional.")
		case errors.Is(err, ErrEmptySettingsPatch):
			writeError(c, http.StatusBadRequest, "invalid_request", "Informe ao menos um campo para atualizar.")
		default:
			var validationError *SettingsFieldError
			if errors.As(err, &validationError) {
				writeFieldError(c, http.StatusBadRequest, validationError.Field, validationError.Code, validationError.Message)
				return
			}
			slog.Error("establishment_settings_update_failed", "request_id", httpserver.RequestID(c))
			writeError(c, http.StatusInternalServerError, "internal_error", "Não foi possível atualizar as configurações do estabelecimento.")
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func decodeSettingsPatch(request settingsPatchRequest) (SettingsPatch, *SettingsFieldError) {
	patch := SettingsPatch{}
	fields := []struct {
		name        string
		raw         json.RawMessage
		nullable    bool
		invalidCode string
		set         func(PatchField[string])
	}{
		{name: "name", raw: request.Name, invalidCode: "invalid_name", set: func(value PatchField[string]) { patch.Name = value }},
		{name: "time_zone", raw: request.TimeZone, invalidCode: "invalid_time_zone", set: func(value PatchField[string]) { patch.TimeZone = value }},
		{name: "description", raw: request.Description, nullable: true, invalidCode: "invalid_text", set: func(value PatchField[string]) { patch.Description = value }},
		{name: "address", raw: request.Address, nullable: true, invalidCode: "invalid_text", set: func(value PatchField[string]) { patch.Address = value }},
		{name: "contact_phone", raw: request.ContactPhone, nullable: true, invalidCode: "invalid_phone", set: func(value PatchField[string]) { patch.ContactPhone = value }},
	}
	for _, field := range fields {
		value, err := decodeSettingsPatchField[string](field.raw, field.name, field.nullable, field.invalidCode)
		if err != nil {
			return SettingsPatch{}, err
		}
		field.set(value)
	}
	linksField, fieldError := decodeSettingsPatchField[[]api.EstablishmentSocialLinkInput](request.SocialLinks, "social_links", true, "invalid_value")
	if fieldError != nil {
		return SettingsPatch{}, fieldError
	}
	patch.SocialLinks = PatchField[[]SocialLink]{Set: linksField.Set, Null: linksField.Null}
	if linksField.Set && !linksField.Null {
		links := make([]SocialLink, 0, len(linksField.Value))
		for _, link := range linksField.Value {
			links = append(links, SocialLink{Label: link.Label, URL: link.Url})
		}
		patch.SocialLinks.Value = links
	}
	minimumNotice, fieldError := decodeSettingsPatchField[int32](request.MinimumBookingNoticeMinutes, "minimum_booking_notice_minutes", true, "invalid_limit")
	if fieldError != nil {
		return SettingsPatch{}, fieldError
	}
	patch.MinimumBookingNoticeMinutes = minimumNotice
	horizon, fieldError := decodeSettingsPatchField[int32](request.BookingHorizonDays, "booking_horizon_days", true, "invalid_horizon")
	if fieldError != nil {
		return SettingsPatch{}, fieldError
	}
	patch.BookingHorizonDays = horizon
	changeDeadline, fieldError := decodeSettingsPatchField[int32](request.CustomerChangeDeadlineMinutes, "customer_change_deadline_minutes", true, "invalid_limit")
	if fieldError != nil {
		return SettingsPatch{}, fieldError
	}
	patch.CustomerChangeDeadlineMinutes = changeDeadline
	return patch, nil
}

func decodeSettingsPatchField[T any](raw json.RawMessage, field string, nullable bool, invalidCode string) (PatchField[T], *SettingsFieldError) {
	if len(raw) == 0 {
		return PatchField[T]{}, nil
	}
	value := PatchField[T]{Set: true}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if !nullable {
			return PatchField[T]{}, invalidSettingsField(field, invalidCode, "Este campo não pode ser removido.")
		}
		value.Null = true
		return value, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value.Value); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return PatchField[T]{}, invalidSettingsField(field, invalidCode, "Verifique o valor informado.")
	}
	return value, nil
}

func writeFieldError(c *gin.Context, status int, field, code, message string) {
	fields := []api.FieldError{{Field: field, Code: code, Message: message}}
	topCode := "validation_failed"
	if status == http.StatusConflict {
		topCode = "conflict"
	}
	c.JSON(status, api.ErrorResponse{Code: topCode, Message: "Verifique os campos informados.", RequestId: httpserver.RequestID(c), Fields: &fields})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}
