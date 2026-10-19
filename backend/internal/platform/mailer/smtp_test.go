package mailer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSMTPTransportRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		address string
		from    string
	}{
		{address: "", from: "Barbershop <no-reply@example.com>"},
		{address: "mailpit:invalid", from: "Barbershop <no-reply@example.com>"},
		{address: "mailpit:1025", from: "invalid"},
	} {
		if _, err := NewSMTPTransport(test.address, test.from); err == nil {
			t.Errorf("NewSMTPTransport(%q, %q) = nil error", test.address, test.from)
		}
	}
}

func TestSMTPTransportRejectsUnsafeMessageWithoutLeakingIt(t *testing.T) {
	transport, err := NewSMTPTransport("mailpit:1025", "Barbershop <no-reply@example.com>")
	if err != nil {
		t.Fatalf("NewSMTPTransport: %v", err)
	}

	err = transport.Send(context.Background(), Message{
		To: "owner@example.com\r\nBcc: leaked@example.com", Subject: "Confirme seu e-mail", Text: "opaque-token",
	})
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("error = %v, want ErrDeliveryFailed", err)
	}
	if strings.Contains(err.Error(), "owner@example.com") || strings.Contains(err.Error(), "opaque-token") {
		t.Fatalf("error leaked message data: %v", err)
	}
}
