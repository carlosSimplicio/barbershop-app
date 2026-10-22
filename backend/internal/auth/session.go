package auth

import (
	"context"
	"errors"
	"time"

	"barbershop-app/backend/internal/auth/security"

	"github.com/google/uuid"
)

var (
	ErrInvalidSession       = errors.New("invalid authentication session")
	ErrPasswordChangeDenied = errors.New("password change is not allowed")
	ErrPasswordUnchanged    = errors.New("new password must differ from current password")
)

type SessionStore interface {
	FindUserByID(context.Context, string) (User, error)
	RevokeSession(context.Context, string, string, time.Time) error
	ChangeTemporaryPassword(context.Context, string, int64, string, string, time.Time, time.Time) error
}

type AccessTokenManager interface {
	AccessTokenIssuer
	Verify(string, time.Time) (security.AccessClaims, error)
}

type PasswordManager interface {
	PasswordVerifier
	Hash(string) (string, error)
}

type SessionService struct {
	store        SessionStore
	passwords    PasswordManager
	accessTokens AccessTokenManager
	csrfTokens   SignedTokenIssuer
	now          func() time.Time
}

func NewSessionService(store SessionStore, passwords PasswordManager, accessTokens AccessTokenManager, csrfTokens SignedTokenIssuer) *SessionService {
	return &SessionService{store: store, passwords: passwords, accessTokens: accessTokens, csrfTokens: csrfTokens, now: time.Now}
}

func (service *SessionService) Logout(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return nil
	}
	now := service.now().UTC()
	claims, err := service.accessTokens.Verify(accessToken, now)
	if err != nil {
		return nil
	}
	err = service.store.RevokeSession(ctx, claims.SessionID, claims.Subject, now)
	if errors.Is(err, ErrInvalidSession) {
		return nil
	}
	return err
}

func (service *SessionService) ChangeTemporaryPassword(ctx context.Context, principal Principal, newPassword string) (LoginResult, error) {
	if err := security.ValidatePassword(newPassword); err != nil {
		return LoginResult{}, err
	}
	user, err := service.store.FindUserByID(ctx, principal.UserID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return LoginResult{}, err
	}
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
	now := service.now().UTC()
	sessionID := uuid.NewString()
	previousVersion := user.CredentialVersion
	user.CredentialVersion++
	user.PasswordChangeRequired = false
	result, err := service.Issue(user, sessionID, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err := service.store.ChangeTemporaryPassword(ctx, user.ID, previousVersion, passwordHash, sessionID, now, now.Add(15*24*time.Hour)); err != nil {
		return LoginResult{}, err
	}
	return result, nil
}

func (service *SessionService) Issue(user User, sessionID string, now time.Time) (LoginResult, error) {
	now = now.UTC()
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
	return LoginResult{
		Principal: Principal{
			UserID: user.ID, EstablishmentID: user.EstablishmentID, Email: user.Email,
			Role: user.Role, PasswordChangeRequired: user.PasswordChangeRequired,
		},
		AccessToken: accessToken, CSRFToken: csrfToken,
	}, nil
}

func userCanAuthenticate(user User) bool {
	return user.Active && (user.Role != "owner" || user.EmailConfirmed)
}
