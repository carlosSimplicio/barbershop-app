package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"barbershop-app/backend/internal/auth/security"
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
	FindUserByNormalizedEmail(context.Context, string) (User, error)
	ListLoginAttempts(context.Context, string, []byte, time.Time) ([]time.Time, error)
	RecordLoginFailure(context.Context, []byte, []byte, time.Time) error
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
	rateKeys          RateKeyGenerator
	dummyPasswordHash string
	now               func() time.Time
}

func NewLoginService(store LoginStore, passwords PasswordVerifier, accessTokens AccessTokenIssuer, csrfTokens SignedTokenIssuer, rateKeys RateKeyGenerator, dummyPasswordHash string) *LoginService {
	return &LoginService{
		store: store, passwords: passwords, accessTokens: accessTokens, csrfTokens: csrfTokens,
		rateKeys: rateKeys, dummyPasswordHash: dummyPasswordHash, now: time.Now,
	}
}

func (service *LoginService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := service.now().UTC()
	normalizedEmail := NormalizeEmail(input.Email)
	ipKey := service.rateKeys.Key("ip", input.IP)
	emailKey := service.rateKeys.Key("email", normalizedEmail)

	retryAfter, err := service.retryAfter(ctx, ipKey, emailKey, now)
	if err != nil {
		return LoginResult{}, err
	}
	if retryAfter > 0 {
		return LoginResult{}, RateLimitError{RetryAfter: retryAfter}
	}

	user, err := service.store.FindUserByNormalizedEmail(ctx, normalizedEmail)
	if errors.Is(err, ErrUserNotFound) {
		_, _ = service.passwords.Verify(input.Password, service.dummyPasswordHash)
		if recordErr := service.store.RecordLoginFailure(ctx, ipKey, emailKey, now); recordErr != nil {
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
		if err := service.store.RecordLoginFailure(ctx, ipKey, emailKey, now); err != nil {
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

func (service *LoginService) retryAfter(ctx context.Context, ipKey, emailKey []byte, now time.Time) (time.Duration, error) {
	var longest time.Duration
	for dimension, key := range map[string][]byte{"ip": ipKey, "email": emailKey} {
		attempts, err := service.store.ListLoginAttempts(ctx, dimension, key, now.Add(-24*time.Hour))
		if err != nil {
			return 0, err
		}
		if duration := rateLimitDuration(attempts, now); duration > longest {
			longest = duration
		}
	}
	return longest, nil
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (err RateLimitError) Error() string { return "authentication rate limited" }

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func rateLimitDuration(attempts []time.Time, now time.Time) time.Duration {
	rules := []struct {
		threshold int
		window    time.Duration
	}{{5, 15 * time.Minute}, {10, time.Hour}, {20, 24 * time.Hour}}
	var longest time.Duration
	for _, rule := range rules {
		var recent []time.Time
		cutoff := now.Add(-rule.window)
		for _, attempt := range attempts {
			if !attempt.Before(cutoff) {
				recent = append(recent, attempt)
			}
		}
		if len(recent) < rule.threshold {
			continue
		}
		boundary := recent[rule.threshold-1]
		retryAfter := boundary.Add(rule.window).Sub(now)
		if retryAfter > longest {
			longest = retryAfter
		}
	}
	return longest
}
