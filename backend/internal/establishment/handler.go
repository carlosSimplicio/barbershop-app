package establishment

import (
	"context"
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

type Handler struct {
	finder Finder
}

func NewHandler(finder Finder) *Handler {
	return &Handler{finder: finder}
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

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorResponse{Code: code, Message: message, RequestId: httpserver.RequestID(c)})
}
