package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

func New(address string, logger *slog.Logger, handler api.ServerInterface, trustedProxies []string, middleware ...gin.HandlerFunc) *http.Server {
	router := gin.New()
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		panic("configure trusted proxies: " + err.Error())
	}
	router.Use(RequestContext(), AccessLog(logger), Recovery(logger))
	router.Use(middleware...)
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
