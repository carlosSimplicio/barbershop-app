package auth

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/ratelimit"

	"github.com/google/uuid"
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
	CredentialVersion      int64
}

type LoginInput struct {
	Email    string
	Password string
	IP       string
}

type LoginResult struct {
	Principal   Principal
	AccessToken string
	CSRFToken   string
}

type LoginStore interface {
	FindUserByNormalizedEmail(context.Context, string) (User, error)
	CreateSession(context.Context, User, string, time.Time) error
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

func NewLoginService(store LoginStore, rateStore ratelimit.Store, passwords PasswordVerifier, accessTokens AccessTokenIssuer, csrfTokens SignedTokenIssuer, rateKeys RateKeyGenerator, dummyPasswordHash string) *LoginService {
	return &LoginService{
		store: store, passwords: passwords, accessTokens: accessTokens, csrfTokens: csrfTokens,
		rateLimiter: ratelimit.New(rateStore, rateKeys), dummyPasswordHash: dummyPasswordHash, now: time.Now,
	}
}

func (service *LoginService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := service.now().UTC()
	normalizedEmail := NormalizeEmail(input.Email)
	identifiers := map[string]string{"ip": input.IP, "email": normalizedEmail}
	reservation, err := service.rateLimiter.Reserve(ctx, "login", identifiers, now)
	if err != nil {
		return LoginResult{}, err
	}

	user, err := service.store.FindUserByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, ErrUserNotFound) {
		_, _ = service.passwords.Verify(input.Password, service.dummyPasswordHash)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, service.releaseRateLimitReservation(ctx, reservation, err)
	}

	passwordMatches, err := service.passwords.Verify(input.Password, user.PasswordHash)
	if err != nil {
		return LoginResult{}, service.releaseRateLimitReservation(ctx, reservation, err)
	}
	if !passwordMatches || !user.Active {
		return LoginResult{}, ErrInvalidCredentials
	}
	if user.Role == "owner" && !user.EmailConfirmed {
		return LoginResult{}, service.releaseRateLimitReservation(ctx, reservation, ErrEmailConfirmationRequired)
	}
	if err := service.releaseRateLimitReservation(ctx, reservation, nil); err != nil {
		return LoginResult{}, err
	}

	sessionID := uuid.NewString()
	accessToken, err := service.accessTokens.Issue(security.Principal{
		UserID: user.ID, SessionID: sessionID, EstablishmentID: user.EstablishmentID, Role: user.Role, CredentialVersion: user.CredentialVersion,
	}, now)
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, err := service.csrfTokens.NewSignedToken()
	if err != nil {
		return LoginResult{}, err
	}
	if err := service.store.CreateSession(ctx, user, sessionID, now.Add(SessionLifetime)); err != nil {
		if errors.Is(err, ErrInvalidSession) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}

	return LoginResult{
		Principal: Principal{
			UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email,
			Role: user.Role, PasswordChangeRequired: user.PasswordChangeRequired, CredentialVersion: user.CredentialVersion,
		},
		AccessToken: accessToken, CSRFToken: csrfToken,
	}, nil
}

func (service *LoginService) releaseRateLimitReservation(ctx context.Context, reservation *ratelimit.Reservation, cause error) error {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := service.rateLimiter.Release(releaseCtx, reservation); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

type RateLimitError = ratelimit.Error
