package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

func New(address string, logger *slog.Logger, handler api.ServerInterface) *http.Server {
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		panic("disable trusted proxies: " + err.Error())
	}
	router.Use(RequestContext(), AccessLog(logger), Recovery(logger))
	api.RegisterHandlersWithOptions(router, handler, api.GinServerOptions{BaseURL: "/api/v1"})

	return &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
