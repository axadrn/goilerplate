package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendMailer struct {
	client    resendEmailSender
	fromEmail string
}

type resendEmailSender interface {
	SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

func NewResendMailer(apiKey, fromEmail string) *ResendMailer {
	client := resend.NewClient(apiKey)
	return &ResendMailer{client: client.Emails, fromEmail: fromEmail}
}

func (m *ResendMailer) Name() string {
	return ProviderResend
}

func (m *ResendMailer) Send(ctx context.Context, message Message) error {
	_, err := m.client.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    m.fromEmail,
		To:      []string{message.To},
		Subject: message.Subject,
		Text:    message.Text,
		Html:    message.HTML,
	})
	if err != nil {
		return fmt.Errorf("send email with Resend: %w", err)
	}
	return nil
}
