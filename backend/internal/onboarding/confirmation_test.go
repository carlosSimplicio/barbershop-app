package onboarding

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth/security"
	"barbershop-app/backend/internal/platform/mailer"
)

func TestNewEmailConfirmationCreatesOpaqueTokenValidForExactly24Hours(t *testing.T) {
	now := time.Date(2026, time.October, 18, 12, 0, 0, 0, time.FixedZone("test", -3*60*60))

	confirmation, err := NewEmailConfirmation(now)
	if err != nil {
		t.Fatalf("NewEmailConfirmation: %v", err)
	}
	if len(confirmation.Token) != 43 {
		t.Fatalf("token length = %d, want 43 base64url characters", len(confirmation.Token))
	}
	if !bytes.Equal(confirmation.TokenHash, security.HashOpaqueToken(confirmation.Token)) {
		t.Fatal("persisted token hash does not match the opaque token")
	}
	wantExpiry := now.UTC().Add(24 * time.Hour)
	if !confirmation.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("ExpiresAt = %s, want %s", confirmation.ExpiresAt, wantExpiry)
	}
}

func TestConfirmationMailerUsesFragmentAndPtBRTemplate(t *testing.T) {
	sender := &senderStub{}
	service := NewConfirmationMailer(sender, "https://localhost:8443")

	if err := service.Send(context.Background(), "owner@example.com", "opaque-token"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sender.message.To != "owner@example.com" {
		t.Fatalf("recipient = %q", sender.message.To)
	}
	if sender.message.Subject != "Confirme seu e-mail" {
		t.Fatalf("subject = %q", sender.message.Subject)
	}
	if !strings.Contains(sender.message.Text, "Confirme seu e-mail") {
		t.Fatal("message is not rendered in pt-BR")
	}
	if !strings.Contains(sender.message.Text, "https://localhost:8443/confirmar-email#token=opaque-token") {
		t.Fatalf("message does not contain confirmation fragment: %q", sender.message.Text)
	}
	if strings.Contains(sender.message.Text, "?token=") {
		t.Fatal("message exposes the token in the query string")
	}
}

func TestConfirmationMailerDoesNotLeakRecipientOrTokenOnFailure(t *testing.T) {
	sender := &senderStub{err: errors.New("provider mentioned owner@example.com and opaque-token")}
	service := NewConfirmationMailer(sender, "https://localhost:8443")

	err := service.Send(context.Background(), "owner@example.com", "opaque-token")
	if !errors.Is(err, mailer.ErrDeliveryFailed) {
		t.Fatalf("error = %v, want ErrDeliveryFailed", err)
	}
	if strings.Contains(err.Error(), "owner@example.com") || strings.Contains(err.Error(), "opaque-token") {
		t.Fatalf("error leaked sensitive data: %v", err)
	}
}

func TestConfirmationServiceReplacesStoredTokenBeforeSending(t *testing.T) {
	store := &confirmationStoreStub{}
	sender := &senderStub{}
	service := NewConfirmationService(store, NewConfirmationMailer(sender, "https://localhost:8443"))

	if err := service.Request(context.Background(), "user-1", "owner@example.com"); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if store.userID != "user-1" {
		t.Fatalf("stored confirmation = %#v", store)
	}
	if sender.message.To != "owner@example.com" || !strings.Contains(sender.message.Text, "/confirmar-email#token=stored-token") {
		t.Fatalf("message was not sent after persistence: %#v", sender.message)
	}
}

type confirmationStoreStub struct {
	userID string
}

func (stub *confirmationStoreStub) ReplaceEmailConfirmation(_ context.Context, userID string) (EmailConfirmation, error) {
	stub.userID = userID
	return EmailConfirmation{Token: "stored-token"}, nil
}

type senderStub struct {
	message mailer.Message
	err     error
}

func (sender *senderStub) Send(_ context.Context, message mailer.Message) error {
	sender.message = message
	return sender.err
}
