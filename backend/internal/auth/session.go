package auth

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth/security"
)

var (
	ErrInvalidSession       = errors.New("invalid authentication session")
	ErrRefreshTokenReused   = errors.New("refresh token reused")
	ErrPasswordChangeDenied = errors.New("password change is not allowed")
	ErrPasswordUnchanged    = errors.New("new password must differ from temporary password")
)

type SessionStore interface {
	FindUserByID(context.Context, string) (User, error)
	RotateRefreshToken(context.Context, []byte, []byte, time.Time, time.Time) (User, error)
	RevokeSessionByRefreshToken(context.Context, []byte, time.Time) error
	ChangeTemporaryPassword(context.Context, string, int64, string, []byte, time.Time, time.Time) (int64, error)
}

type PasswordManager interface {
	PasswordVerifier
	Hash(string) (string, error)
}

type SessionService struct {
	store        SessionStore
	passwords    PasswordManager
	accessTokens AccessTokenIssuer
	csrfTokens   SignedTokenIssuer
	now          func() time.Time
}

func NewSessionService(store SessionStore, passwords PasswordManager, accessTokens AccessTokenIssuer, csrfTokens SignedTokenIssuer) *SessionService {
	return &SessionService{store: store, passwords: passwords, accessTokens: accessTokens, csrfTokens: csrfTokens, now: time.Now}
}

func (service *SessionService) Refresh(ctx context.Context, currentToken string) (LoginResult, error) {
	if currentToken == "" {
		return LoginResult{}, ErrInvalidSession
	}
	newRefreshToken, err := security.NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := service.now().UTC()
	user, err := service.store.RotateRefreshToken(ctx, security.HashOpaqueToken(currentToken), security.HashOpaqueToken(newRefreshToken), now, now.Add(7*24*time.Hour))
	if err != nil {
		return LoginResult{}, err
	}
	if !userCanAuthenticate(user) {
		return LoginResult{}, ErrInvalidSession
	}
	return service.issueResult(user, newRefreshToken, now)
}

func (service *SessionService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return service.store.RevokeSessionByRefreshToken(ctx, security.HashOpaqueToken(refreshToken), service.now().UTC())
}

func (service *SessionService) ChangeTemporaryPassword(ctx context.Context, principal Principal, newPassword string) (LoginResult, error) {
	if err := security.ValidatePassword(newPassword); err != nil {
		return LoginResult{}, err
	}
	user, err := service.store.FindUserByID(ctx, principal.UserID)
	if err != nil || !userCanAuthenticate(user) || !user.PasswordChangeRequired {
		return LoginResult{}, ErrPasswordChangeDenied
	}
	unchanged, err := service.passwords.Verify(newPassword, user.PasswordHash)
	if err != nil {
		return LoginResult{}, err
	}
	if unchanged {
		return LoginResult{}, ErrPasswordUnchanged
	}
	passwordHash, err := service.passwords.Hash(newPassword)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, err := security.NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := service.now().UTC()
	credentialVersion, err := service.store.ChangeTemporaryPassword(
		ctx, user.ID, user.CredentialVersion, passwordHash, security.HashOpaqueToken(refreshToken), now, now.Add(7*24*time.Hour),
	)
	if err != nil {
		return LoginResult{}, err
	}
	user.CredentialVersion = credentialVersion
	user.PasswordChangeRequired = false
	return service.issueResult(user, refreshToken, now)
}

func (service *SessionService) issueResult(user User, refreshToken string, now time.Time) (LoginResult, error) {
	accessToken, err := service.accessTokens.Issue(security.Principal{
		UserID: user.ID, EstablishmentID: user.EstablishmentID, Role: user.Role, CredentialVersion: user.CredentialVersion,
	}, now)
	if err != nil {
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

func userCanAuthenticate(user User) bool {
	return user.Active && (user.Role != "owner" || user.EmailConfirmed)
}
