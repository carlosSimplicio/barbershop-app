package onboarding

import (
	"bytes"
	"context"
	_ "embed"
	"strings"
	"text/template"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/mailer"
)

const emailConfirmationLifetime = 24 * time.Hour

//go:embed templates/email_confirmation.pt-BR.txt
var emailConfirmationTemplateSource string

var emailConfirmationTemplate = template.Must(template.New("email-confirmation-pt-BR").Parse(emailConfirmationTemplateSource))

type EmailConfirmation struct {
	Token     string
	TokenHash []byte
	ExpiresAt time.Time
}

func NewEmailConfirmation(now time.Time) (EmailConfirmation, error) {
	token, err := security.NewOpaqueToken()
	if err != nil {
		return EmailConfirmation{}, err
	}
	return EmailConfirmation{
		Token: token, TokenHash: security.HashOpaqueToken(token), ExpiresAt: now.UTC().Add(emailConfirmationLifetime),
	}, nil
}

type ConfirmationMailer struct {
	sender  mailer.Sender
	baseURL string
}

type ConfirmationStore interface {
	ReplaceEmailConfirmation(context.Context, string) (EmailConfirmation, error)
}

type ConfirmationService struct {
	store  ConfirmationStore
	mailer *ConfirmationMailer
}

func NewConfirmationService(store ConfirmationStore, confirmationMailer *ConfirmationMailer) *ConfirmationService {
	return &ConfirmationService{store: store, mailer: confirmationMailer}
}

func (service *ConfirmationService) Request(ctx context.Context, userID, recipient string) error {
	confirmation, err := service.store.ReplaceEmailConfirmation(ctx, userID)
	if err != nil {
		return err
	}
	return service.mailer.Send(ctx, recipient, confirmation.Token)
}

func NewConfirmationMailer(sender mailer.Sender, baseURL string) *ConfirmationMailer {
	return &ConfirmationMailer{sender: sender, baseURL: baseURL}
}

func (service *ConfirmationMailer) Send(ctx context.Context, recipient, token string) error {
	confirmationURL := strings.TrimRight(service.baseURL, "/") + "/confirmar-email#token=" + token
	var body bytes.Buffer
	if err := emailConfirmationTemplate.Execute(&body, struct{ ConfirmationURL string }{confirmationURL}); err != nil {
		return mailer.ErrDeliveryFailed
	}
	if err := service.sender.Send(ctx, mailer.Message{
		To: recipient, Subject: "Confirme seu e-mail", Text: body.String(),
	}); err != nil {
		return mailer.ErrDeliveryFailed
	}
	return nil
}
