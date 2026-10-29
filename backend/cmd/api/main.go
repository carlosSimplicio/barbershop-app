package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/password_recovery"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/config"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/establishment"
	"barbershop-app/backend/internal/health"
	"barbershop-app/backend/internal/httpapi"
	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/onboarding/email_confirmation"
	"barbershop-app/backend/internal/onboarding/registration"
	"barbershop-app/backend/internal/platform/database"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
	"barbershop-app/backend/internal/platform/storage"
	"barbershop-app/backend/internal/professionals"

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
	logoStorage, err := storage.NewLocalStore(configuration.LogoStorageDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := logoStorage.Close(); err != nil {
			logger.Error("logo_storage_close_failed")
		}
	}()
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

	passwords, err := security.NewPasswordHasher(security.PasswordParameters{
		MemoryKiB: configuration.Argon2MemoryKiB, Iterations: configuration.Argon2Iterations,
		Parallelism: configuration.Argon2Parallelism, SaltLength: 16, KeyLength: 32,
	})
	if err != nil {
		return err
	}
	accessTokens, err := security.NewJWTManager(configuration.JWTActiveKeyID, configuration.JWTPrivateKey, configuration.JWTPublicKeys, auth.SessionLifetime)
	if err != nil {
		return err
	}
	csrfTokens, err := security.NewHMACSigner(configuration.CSRFSigningKey)
	if err != nil {
		return err
	}
	rateKeys, err := security.NewHMACSigner(configuration.RateLimitHMACKey)
	if err != nil {
		return err
	}
	dummyPasswordHash, err := passwords.Hash("invalid-password-probe")
	if err != nil {
		return err
	}
	authStore := auth.NewStore(pool)
	onboardingStore := onboarding.NewStore(pool)
	rateLimitStore := ratelimit.NewPostgresStore(pool)
	var emailSender mailer.Sender = disabledEmailSender{}
	if configuration.EmailProvider == "smtp" {
		emailSender, err = mailer.NewSMTPTransport(net.JoinHostPort(configuration.SMTPHost, strconv.Itoa(int(configuration.SMTPPort))), configuration.SMTPFrom)
		if err != nil {
			return err
		}
	}
	confirmationMailer := emailconfirmation.NewConfirmationMailer(emailSender, configuration.PublicAppURL)
	confirmationService := emailconfirmation.NewConfirmationService(onboardingStore, confirmationMailer)
	confirmationRequests := emailconfirmation.NewEmailConfirmationRequestService(onboardingStore, confirmationService, rateLimitStore, rateKeys)
	registrations := registration.NewRegistrationService(onboardingStore, rateLimitStore, passwords, confirmationMailer, rateKeys)
	login := auth.NewLoginService(authStore, rateLimitStore, passwords, accessTokens, csrfTokens, rateKeys, dummyPasswordHash)
	sessions := auth.NewSessionService(authStore, passwords, accessTokens, csrfTokens)
	confirmationVerification := emailconfirmation.NewEmailConfirmationVerificationService(onboardingStore, sessions)
	writeEntitlements := entitlements.NewService(entitlements.NewPostgresStore(pool))
	establishments := establishment.NewService(establishment.NewPostgresStore(pool))
	establishmentSettings := establishment.NewSettingsService(establishment.NewPostgresSettingsStore(pool), writeEntitlements)
	establishmentLogoUploads := establishment.NewLogoUploadService(establishment.NewPostgresLogoUploadStore(pool), logoStorage, writeEntitlements)
	publicEstablishmentStore := establishment.NewPostgresPublicEstablishmentStore(pool)
	publicEstablishmentResolver := establishment.NewPublicEstablishmentResolver(publicEstablishmentStore)
	publicEstablishments := establishment.NewPublicEstablishmentService(publicEstablishmentResolver, publicEstablishmentStore)
	publicEstablishmentLogos := establishment.NewPublicLogoService(publicEstablishmentResolver, establishment.NewPostgresPublicLogoStore(pool), logoStorage)
	professionalService := professionals.NewService(professionals.NewPostgresStore(pool), passwords, writeEntitlements)
	passwordRecovery := passwordrecovery.NewPasswordRecoveryRequestService(authStore, rateLimitStore, rateKeys, emailSender, configuration.PublicAppURL)
	passwordRecoveryVerification := passwordrecovery.NewPasswordRecoveryVerificationService(authStore, passwords)
	handler := httpapi.NewHandler(
		health.NewHandler(pool),
		auth.NewHandler(login, sessions, passwordRecovery, passwordRecoveryVerification),
		onboarding.NewHandler(registrations, confirmationRequests, confirmationVerification),
		establishment.NewHandler(establishments, establishmentSettings, establishmentLogoUploads, publicEstablishments, publicEstablishmentLogos),
		professionals.NewHandler(professionalService),
	)
	authMiddleware := auth.NewMiddleware(authStore, accessTokens, csrfTokens, configuration.TrustedOrigins)
	server := httpserver.New(configuration.HTTPAddress, logger, handler, configuration.TrustedProxyCIDRs, authMiddleware.Handle())
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

type disabledEmailSender struct{}

func (disabledEmailSender) Send(context.Context, mailer.Message) error {
	return mailer.ErrDeliveryFailed
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
