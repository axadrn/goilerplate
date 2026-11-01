package mailer

import (
	"context"
	"fmt"
	"time"

	mail "github.com/wneessen/go-mail"
	"{{.ModulePath}}/internal/config"
)

const smtpTimeout = 10 * time.Second

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string
	From     string
}

type smtpSender interface {
	DialAndSendWithContext(ctx context.Context, messages ...*mail.Msg) error
}

type SMTPMailer struct {
	client    smtpSender
	fromEmail string
}

func NewSMTPMailer(cfg SMTPConfig) (*SMTPMailer, error) {
	options := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithTimeout(smtpTimeout),
	}

	switch cfg.TLS {
	case config.SMTPTLSStartTLS:
		options = append(options, mail.WithTLSPolicy(mail.TLSMandatory))
	case config.SMTPTLSImplicit:
		options = append(options, mail.WithSSL())
	case config.SMTPTLSNone:
		options = append(options, mail.WithTLSPolicy(mail.NoTLS))
	default:
		return nil, fmt.Errorf("unsupported SMTP TLS mode: %s", cfg.TLS)
	}

	if cfg.Username != "" {
		options = append(options,
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	}

	client, err := mail.NewClient(cfg.Host, options...)
	if err != nil {
		return nil, fmt.Errorf("create SMTP client: %w", err)
	}

	return &SMTPMailer{client: client, fromEmail: cfg.From}, nil
}

func (m *SMTPMailer) Name() string {
	return ProviderSMTP
}

func (m *SMTPMailer) Send(ctx context.Context, message Message) error {
	mailMessage := mail.NewMsg()
	if err := mailMessage.From(m.fromEmail); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := mailMessage.To(message.To); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	mailMessage.Subject(message.Subject)
	mailMessage.SetBodyString(mail.TypeTextPlain, message.Text)
	if message.HTML != "" {
		mailMessage.AddAlternativeString(mail.TypeTextHTML, message.HTML)
	}

	if err := m.client.DialAndSendWithContext(ctx, mailMessage); err != nil {
		return fmt.Errorf("send email with SMTP: %w", err)
	}
	return nil
}
