//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/auth/password_recovery"
	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/entitlements"
	"barbershop-app/backend/internal/establishment"
	"barbershop-app/backend/internal/health"
	"barbershop-app/backend/internal/httpapi"
	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/onboarding/email_confirmation"
	"barbershop-app/backend/internal/onboarding/registration"
	"barbershop-app/backend/internal/platform/httpserver"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
	"barbershop-app/backend/internal/platform/storage"
	"barbershop-app/backend/internal/professionals"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	trustedOrigin    = "https://app.example.test"
	accessCookieName = "__Secure-barbershop_access"
	csrfCookieName   = "__Secure-barbershop_csrf"
)

var integrationPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := postgrescontainer.Run(ctx, "postgres:18.6-alpine3.24",
		postgrescontainer.WithDatabase("barbershop_test"),
		postgrescontainer.WithUsername("barbershop"),
		postgrescontainer.WithPassword("integration-test-password"),
		postgrescontainer.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start integration PostgreSQL: %v\n", err)
		os.Exit(1)
	}

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		fmt.Fprintf(os.Stderr, "build integration database URL: %v\n", err)
		os.Exit(1)
	}
	if err := migrateDatabase(ctx, connectionString); err != nil {
		_ = testcontainers.TerminateContainer(container)
		fmt.Fprintf(os.Stderr, "migrate integration database: %v\n", err)
		os.Exit(1)
	}
	integrationPool, err = pgxpool.New(ctx, connectionString)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		fmt.Fprintf(os.Stderr, "open integration database pool: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	integrationPool.Close()
	if err := testcontainers.TerminateContainer(container); err != nil {
		fmt.Fprintf(os.Stderr, "terminate integration PostgreSQL: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func migrateDatabase(ctx context.Context, connectionString string) error {
	database, err := sql.Open("pgx", connectionString)
	if err != nil {
		return err
	}
	defer database.Close()

	_, currentFile, _, _ := runtime.Caller(0)
	migrationDirectory := filepath.Join(filepath.Dir(currentFile), "..", "db", "migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, database, migrationDirectory)
}

type testApplication struct {
	server       *httptest.Server
	client       *http.Client
	messages     *messageCapture
	passwords    *security.PasswordHasher
	accessTokens *security.JWTManager
	logoStorage  *storage.LocalStore
}

func newTestApplication(t *testing.T) *testApplication {
	t.Helper()
	resetDatabase(t)
	gin.SetMode(gin.TestMode)

	passwords, err := security.NewPasswordHasher(security.PasswordParameters{
		MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	})
	if err != nil {
		t.Fatalf("create password hasher: %v", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate JWT key: %v", err)
	}
	accessTokens, err := security.NewJWTManager("integration-key", privateKey, map[string]ed25519.PublicKey{"integration-key": publicKey}, 15*24*time.Hour)
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	csrfTokens, err := security.NewHMACSigner(bytes.Repeat([]byte{0x31}, 32))
	if err != nil {
		t.Fatalf("create CSRF signer: %v", err)
	}
	rateKeys, err := security.NewHMACSigner(bytes.Repeat([]byte{0x52}, 32))
	if err != nil {
		t.Fatalf("create rate-limit signer: %v", err)
	}
	dummyPasswordHash, err := passwords.Hash("invalid-password-probe")
	if err != nil {
		t.Fatalf("hash dummy password: %v", err)
	}

	authStore := auth.NewStore(integrationPool)
	onboardingStore := onboarding.NewStore(integrationPool)
	rateLimitStore := ratelimit.NewPostgresStore(integrationPool)
	messages := &messageCapture{}
	confirmationMailer := emailconfirmation.NewConfirmationMailer(messages, trustedOrigin)
	sessions := auth.NewSessionService(authStore, passwords, accessTokens, csrfTokens)
	writeEntitlements := entitlements.NewService(entitlements.NewPostgresStore(integrationPool))
	logoStorage, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("create test logo storage: %v", err)
	}
	t.Cleanup(func() { _ = logoStorage.Close() })
	publicEstablishmentStore := establishment.NewPostgresPublicEstablishmentStore(integrationPool)
	handler := httpapi.NewHandler(
		health.NewHandler(integrationPool),
		auth.NewHandler(auth.NewLoginService(authStore, rateLimitStore, passwords, accessTokens, csrfTokens, rateKeys, dummyPasswordHash), sessions,
			passwordrecovery.NewPasswordRecoveryRequestService(authStore, rateLimitStore, rateKeys, messages, trustedOrigin),
			passwordrecovery.NewPasswordRecoveryVerificationService(authStore, passwords)),
		onboarding.NewHandler(
			registration.NewRegistrationService(onboardingStore, rateLimitStore, passwords, confirmationMailer, rateKeys),
			emailconfirmation.NewEmailConfirmationRequestService(onboardingStore, emailconfirmation.NewConfirmationService(onboardingStore, confirmationMailer), rateLimitStore, rateKeys),
			emailconfirmation.NewEmailConfirmationVerificationService(onboardingStore, sessions),
		),
		establishment.NewHandler(
			establishment.NewService(establishment.NewPostgresStore(integrationPool)),
			establishment.NewSettingsService(establishment.NewPostgresSettingsStore(integrationPool), writeEntitlements),
			establishment.NewLogoUploadService(establishment.NewPostgresLogoUploadStore(integrationPool), logoStorage, writeEntitlements),
			establishment.NewPublicEstablishmentService(
				establishment.NewPublicEstablishmentResolver(publicEstablishmentStore), publicEstablishmentStore),
			establishment.NewPublicLogoService(
				establishment.NewPublicEstablishmentResolver(publicEstablishmentStore),
				establishment.NewPostgresPublicLogoStore(integrationPool), logoStorage),
		),
		professionals.NewHandler(professionals.NewService(professionals.NewPostgresStore(integrationPool), passwords,
			writeEntitlements)),
	)
	middleware := auth.NewMiddleware(authStore, accessTokens, csrfTokens, []string{trustedOrigin})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	configuredServer := httpserver.New(":0", logger, handler, nil, middleware.Handle())
	server := httptest.NewTLSServer(configuredServer.Handler)
	jar, err := cookiejar.New(nil)
	if err != nil {
		server.Close()
		t.Fatalf("create cookie jar: %v", err)
	}
	client := server.Client()
	client.Jar = jar

	application := &testApplication{server: server, client: client, messages: messages, passwords: passwords, accessTokens: accessTokens, logoStorage: logoStorage}
	t.Cleanup(server.Close)
	return application
}

func resetDatabase(t *testing.T) {
	t.Helper()
	_, err := integrationPool.Exec(context.Background(), `
		TRUNCATE auth_login_attempts, auth_sessions, establishment_trials,
			email_confirmations, user_credentials, users, establishments RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("reset integration database: %v", err)
	}
}

type messageCapture struct {
	mu       sync.Mutex
	messages []mailer.Message
}

func (capture *messageCapture) Send(_ context.Context, message mailer.Message) error {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.messages = append(capture.messages, message)
	return nil
}

func (capture *messageCapture) latestToken(t *testing.T) string {
	t.Helper()
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if len(capture.messages) == 0 {
		t.Fatal("expected a confirmation e-mail")
	}
	body := capture.messages[len(capture.messages)-1].Text
	marker := "#token="
	index := strings.Index(body, marker)
	if index < 0 {
		t.Fatalf("confirmation e-mail does not contain token fragment: %q", body)
	}
	token := body[index+len(marker):]
	if end := strings.IndexAny(token, " \r\n"); end >= 0 {
		token = token[:end]
	}
	return token
}

func (application *testApplication) request(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, application.server.URL+path, payload)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		request.Header.Set("Origin", trustedOrigin)
	}
	if requiresTestCSRF(method, path) {
		request.Header.Set("X-CSRF-Token", application.cookie(t, csrfCookieName).Value)
	}
	response, err := application.client.Do(request)
	if err != nil {
		t.Fatalf("perform %s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func requiresTestCSRF(method, path string) bool {
	if method == http.MethodGet || method == http.MethodHead {
		return false
	}
	return path == "/api/v1/auth/logout" || path == "/api/v1/auth/password/change" || path == "/api/v1/establishment/settings" || path == "/api/v1/establishment/logo" || path == "/api/v1/professionals"
}

func (application *testApplication) cookie(t *testing.T, name string) *http.Cookie {
	t.Helper()
	serverURL, err := url.Parse(application.server.URL + "/api/v1/auth/logout")
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	for _, cookie := range application.client.Jar.Cookies(serverURL) {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}

func assertStatus(t *testing.T, response *http.Response, expected int) {
	t.Helper()
	if response.StatusCode == expected {
		return
	}
	body, _ := io.ReadAll(response.Body)
	t.Fatalf("expected HTTP %d, got %d: %s", expected, response.StatusCode, body)
}

func decodeJSON(t *testing.T, response *http.Response, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
