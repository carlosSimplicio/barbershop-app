package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"barbershop-app/backend/internal/config"
	"barbershop-app/backend/internal/health"
	"barbershop-app/backend/internal/platform/database"
	"barbershop-app/backend/internal/platform/httpserver"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application_stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(configuration)
	slog.SetDefault(logger)
	if configuration.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(rootContext, configuration.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	server := httpserver.New(configuration.HTTPAddress, logger, health.NewHandler(pool))
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http_server_started", "address", configuration.HTTPAddress)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-rootContext.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), configuration.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownContext)
}

func newLogger(configuration config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch configuration.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	options := &slog.HandlerOptions{Level: level}
	if configuration.Environment == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
