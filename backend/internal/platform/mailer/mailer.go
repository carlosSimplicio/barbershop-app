package mailer

import (
	"context"
	"errors"
)

var ErrDeliveryFailed = errors.New("transactional email delivery failed")

type Message struct {
	To      string
	Subject string
	Text    string
}

type Sender interface {
	Send(context.Context, Message) error
}
