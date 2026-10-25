package emailconfirmation

import (
	"bytes"
	"context"
	_ "embed"
	"strings"
	"text/template"

	"barbershop-app/backend/internal/onboarding"
	"barbershop-app/backend/internal/platform/mailer"
)

//go:embed templates/email_confirmation.pt-BR.txt
var emailConfirmationTemplateSource string

var emailConfirmationTemplate = template.Must(template.New("email-confirmation-pt-BR").Parse(emailConfirmationTemplateSource))

type ConfirmationMailer struct {
	sender  mailer.Sender
	baseURL string
}

type ConfirmationService struct {
	store  onboarding.ConfirmationStore
	mailer *ConfirmationMailer
}

func NewConfirmationService(store onboarding.ConfirmationStore, confirmationMailer *ConfirmationMailer) *ConfirmationService {
	return &ConfirmationService{store: store, mailer: confirmationMailer}
}

func (service *ConfirmationService) Request(ctx context.Context, userID, recipient string) error {
	confirmation, replaced, err := service.store.ReplaceEmailConfirmation(ctx, userID)
	if err != nil {
		return err
	}
	if !replaced {
		return nil
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
