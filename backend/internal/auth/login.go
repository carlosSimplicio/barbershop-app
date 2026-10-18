package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/ratelimit"
)

var (
	ErrUserNotFound              = errors.New("authentication user not found")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrEmailConfirmationRequired = errors.New("email confirmation required")
)

type User struct {
	ID                     string
	EstablishmentID        string
	Email                  string
	NormalizedEmail        string
	Role                   string
	Active                 bool
	EmailConfirmed         bool
	CredentialVersion      int64
	PasswordChangeRequired bool
	PasswordHash           string
}

type Principal struct {
	UserID                 string
	EstablishmentID        string
	Email                  string
	Role                   string
	PasswordChangeRequired bool
}

type LoginInput struct {
	Email    string
	Password string
	IP       string
}

type LoginResult struct {
	Principal    Principal
	AccessToken  string
	RefreshToken string
	CSRFToken    string
}

type LoginStore interface {
	ratelimit.Store
	FindUserByNormalizedEmail(context.Context, string) (User, error)
	CreateSession(context.Context, string, []byte, time.Time) error
}

type PasswordVerifier interface {
	Verify(string, string) (bool, error)
}

type AccessTokenIssuer interface {
	Issue(security.Principal, time.Time) (string, error)
}

type SignedTokenIssuer interface {
	NewSignedToken() (string, error)
}

type RateKeyGenerator interface {
	Key(string, string) []byte
}

type LoginService struct {
	store             LoginStore
	passwords         PasswordVerifier
	accessTokens      AccessTokenIssuer
	csrfTokens        SignedTokenIssuer
	rateLimiter       *ratelimit.Limiter
	dummyPasswordHash string
	now               func() time.Time
}

func NewLoginService(store LoginStore, passwords PasswordVerifier, accessTokens AccessTokenIssuer, csrfTokens SignedTokenIssuer, rateKeys RateKeyGenerator, dummyPasswordHash string) *LoginService {
	return &LoginService{
		store: store, passwords: passwords, accessTokens: accessTokens, csrfTokens: csrfTokens,
		rateLimiter: ratelimit.New(store, rateKeys), dummyPasswordHash: dummyPasswordHash, now: time.Now,
	}
}

func (service *LoginService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := service.now().UTC()
	normalizedEmail := NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	if err := service.rateLimiter.Check(ctx, "login", identifiers, now); err != nil {
		return LoginResult{}, err
	}

	user, err := service.store.FindUserByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, ErrUserNotFound) {
		_, _ = service.passwords.Verify(input.Password, service.dummyPasswordHash)
		if recordErr := service.rateLimiter.Record(ctx, "login", identifiers, now); recordErr != nil {
			return LoginResult{}, recordErr
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	passwordMatches, err := service.passwords.Verify(input.Password, user.PasswordHash)
	if err != nil {
		return LoginResult{}, err
	}
	if !passwordMatches || !user.Active {
		if err := service.rateLimiter.Record(ctx, "login", identifiers, now); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	if user.Role == "owner" && !user.EmailConfirmed {
		return LoginResult{}, ErrEmailConfirmationRequired
	}

	refreshToken, err := security.NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	accessToken, err := service.accessTokens.Issue(security.Principal{
		UserID: user.ID, EstablishmentID: user.EstablishmentID, Role: user.Role, CredentialVersion: user.CredentialVersion,
	}, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err := service.store.CreateSession(ctx, user.ID, security.HashOpaqueToken(refreshToken), now.Add(7*24*time.Hour)); err != nil {
		return LoginResult{}, err
	}
	csrfToken, err := service.csrfTokens.NewSignedToken()
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Principal: Principal{
			UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email,
			Role: user.Role, PasswordChangeRequired: user.PasswordChangeRequired,
		},
		AccessToken: accessToken, RefreshToken: refreshToken, CSRFToken: csrfToken,
	}, nil
}

type RateLimitError = ratelimit.Error

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
