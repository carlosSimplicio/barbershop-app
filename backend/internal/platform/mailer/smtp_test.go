package mailer

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
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

func TestSMTPSendBoundsConnectionEstablishment(t *testing.T) {
	for _, parentDeadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "transport budget", true: "earlier caller deadline"}[parentDeadline], func(t *testing.T) {
			transport, err := NewSMTPTransport("mailpit:1025", "no-reply@example.test")
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			var expected time.Time
			if parentDeadline {
				var cancel context.CancelFunc
				expected = time.Now().Add(time.Second)
				ctx, cancel = context.WithDeadline(ctx, expected)
				defer cancel()
			}
			transport.dialContext = func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				deadline, ok := dialCtx.Deadline()
				if !ok {
					t.Error("connection establishment has no deadline")
				} else if remaining := time.Until(deadline); remaining <= 0 || remaining > smtpOperationTimeout {
					t.Error("connection exceeds the total SMTP budget")
				} else if parentDeadline && !deadline.Equal(expected) {
					t.Error("caller deadline was extended")
				}
				return nil, context.DeadlineExceeded
			}
			err = transport.Send(ctx, Message{To: "owner@example.test", Subject: "Confirmação", Text: "Mensagem de teste"})
			if !errors.Is(err, ErrDeliveryFailed) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSMTPSendCancellationInterruptsAnUnresponsiveServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	transport, err := NewSMTPTransport(listener.Addr().String(), "no-reply@example.test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- transport.Send(ctx, Message{To: "owner@example.test", Subject: "Confirmação", Text: "Mensagem de teste"})
	}()
	if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	connection, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, ErrDeliveryFailed) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt SMTP greeting")
	}
}
