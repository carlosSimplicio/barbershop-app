package mailer

import (
	"bufio"
	"context"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const smtpOperationTimeout = 10 * time.Second

type SMTPTransport struct {
	address     string
	host        string
	from        string
	dialContext func(context.Context, string, string) (net.Conn, error)
}

func NewSMTPTransport(address, from string) (*SMTPTransport, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return nil, fmt.Errorf("SMTP_ADDRESS must use host:port")
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort == 0 {
		return nil, fmt.Errorf("SMTP_ADDRESS must use a valid port")
	}
	parsedFrom, err := parseMailbox(from)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	return &SMTPTransport{address: address, host: host, from: parsedFrom, dialContext: (&net.Dialer{}).DialContext}, nil
}

func (transport *SMTPTransport) Send(ctx context.Context, message Message) error {
	recipient, err := parseMailbox(message.To)
	if err != nil || message.Subject == "" || strings.ContainsAny(message.Subject, "\r\n") {
		return ErrDeliveryFailed
	}
	ctx, cancel := context.WithTimeout(ctx, smtpOperationTimeout)
	defer cancel()
	connection, err := transport.dialContext(ctx, "tcp", transport.address)
	if err != nil {
		return ErrDeliveryFailed
	}
	defer connection.Close()
	stopCancellation := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopCancellation()
	deadline, _ := ctx.Deadline()
	if err := connection.SetDeadline(deadline); err != nil {
		return ErrDeliveryFailed
	}
	client, err := smtp.NewClient(connection, transport.host)
	if err != nil {
		return ErrDeliveryFailed
	}
	defer client.Close()
	if err := client.Mail(transport.from); err != nil {
		return ErrDeliveryFailed
	}
	if err := client.Rcpt(recipient); err != nil {
		return ErrDeliveryFailed
	}
	writer, err := client.Data()
	if err != nil {
		return ErrDeliveryFailed
	}
	buffer := bufio.NewWriter(writer)
	_, writeErr := fmt.Fprintf(buffer,
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		transport.from, recipient, mime.QEncoding.Encode("UTF-8", message.Subject), message.Text,
	)
	flushErr := buffer.Flush()
	closeErr := writer.Close()
	if writeErr != nil || flushErr != nil || closeErr != nil || client.Quit() != nil {
		return ErrDeliveryFailed
	}
	return nil
}

func parseMailbox(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", ErrDeliveryFailed
	}
	address, err := mail.ParseAddress(value)
	if err != nil || !strings.Contains(address.Address, "@") {
		return "", ErrDeliveryFailed
	}
	return address.Address, nil
}
