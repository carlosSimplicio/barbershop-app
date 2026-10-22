package auth

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"strings"
	"text/template"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/mailer"
	"barbershop-app/backend/internal/platform/ratelimit"
)

const passwordRecoveryLifetime = time.Hour

//go:embed templates/password_recovery.pt-BR.txt
var passwordRecoveryTemplateSource string

var passwordRecoveryTemplate = template.Must(template.New("password-recovery-pt-BR").Parse(passwordRecoveryTemplateSource))

type PasswordRecoveryAccount struct {
	UserID string
	Email  string
}

type PasswordRecoveryToken struct {
	Token     string
	TokenHash []byte
	ExpiresAt time.Time
}

type PasswordRecoveryStore interface {
	FindEligibleOwnerForPasswordRecovery(context.Context, string) (PasswordRecoveryAccount, bool, error)
	ReplacePasswordRecoveryToken(context.Context, string) (PasswordRecoveryToken, bool, error)
}

type PasswordRecoveryRequestResult struct {
	DeliveryFailed bool
}

type PasswordRecoveryRequestService struct {
	store       PasswordRecoveryStore
	rateLimiter *ratelimit.Limiter
	sender      mailer.Sender
	baseURL     string
	now         func() time.Time
}

func NewPasswordRecoveryRequestService(store PasswordRecoveryStore, rateStore ratelimit.Store, rateKeys ratelimit.KeyGenerator, sender mailer.Sender, baseURL string) *PasswordRecoveryRequestService {
	return &PasswordRecoveryRequestService{
		store: store, rateLimiter: ratelimit.New(rateStore, rateKeys), sender: sender, baseURL: baseURL, now: time.Now,
	}
}

func (service *PasswordRecoveryRequestService) Request(ctx context.Context, email, ip string) (PasswordRecoveryRequestResult, error) {
	if err := ValidateEmail(email); err != nil {
		return PasswordRecoveryRequestResult{}, err
	}
	normalizedEmail := NormalizeEmail(email)
	if _, err := service.rateLimiter.Reserve(ctx, "password-recovery", map[string]string{"ip": ip, "email": normalizedEmail}, service.now().UTC()); err != nil {
		return PasswordRecoveryRequestResult{}, err
	}
	account, found, err := service.store.FindEligibleOwnerForPasswordRecovery(ctx, normalizedEmail)
	if err != nil || !found {
		return PasswordRecoveryRequestResult{}, err
	}
	token, replaced, err := service.store.ReplacePasswordRecoveryToken(ctx, account.UserID)
	if err != nil || !replaced {
		return PasswordRecoveryRequestResult{}, err
	}
	if err := service.send(ctx, account.Email, token.Token); err != nil {
		if errors.Is(err, mailer.ErrDeliveryFailed) {
			return PasswordRecoveryRequestResult{DeliveryFailed: true}, nil
		}
		return PasswordRecoveryRequestResult{}, err
	}
	return PasswordRecoveryRequestResult{}, nil
}

func (service *PasswordRecoveryRequestService) send(ctx context.Context, recipient, token string) error {
	recoveryURL := strings.TrimRight(service.baseURL, "/") + "/redefinir-senha#token=" + token
	var body bytes.Buffer
	if err := passwordRecoveryTemplate.Execute(&body, struct{ RecoveryURL string }{recoveryURL}); err != nil {
		return mailer.ErrDeliveryFailed
	}
	if err := service.sender.Send(ctx, mailer.Message{
		To: recipient, Subject: "Redefina sua senha", Text: body.String(),
	}); err != nil {
		return mailer.ErrDeliveryFailed
	}
	return nil
}

func newPasswordRecoveryToken(now time.Time) (PasswordRecoveryToken, error) {
	token, err := security.NewOpaqueToken()
	if err != nil {
		return PasswordRecoveryToken{}, err
	}
	return PasswordRecoveryToken{
		Token: token, TokenHash: security.HashOpaqueToken(token), ExpiresAt: now.UTC().Add(passwordRecoveryLifetime),
	}, nil
}
