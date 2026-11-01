package mailer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	mail "github.com/wneessen/go-mail"
)

type smtpSenderStub struct {
	message *mail.Msg
	err     error
}

func (s *smtpSenderStub) DialAndSendWithContext(_ context.Context, messages ...*mail.Msg) error {
	if len(messages) > 0 {
		s.message = messages[0]
	}
	return s.err
}

func TestSMTPMailerBuildsMultipartMessage(t *testing.T) {
	client := &smtpSenderStub{}
	provider := &SMTPMailer{client: client, fromEmail: "from@example.com"}

	err := provider.Send(context.Background(), Message{
		To:      "to@example.com",
		Subject: "Welcome",
		Text:    "Hello from SMTP.",
		HTML:    `<p>Hello from <a href="https://example.com">SMTP</a>.</p>`,
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if client.message == nil {
		t.Fatal("Send() did not call SMTP client")
	}
	if got := client.message.GetFrom(); len(got) != 1 || got[0].Address != "from@example.com" {
		t.Fatalf("From = %v", got)
	}
	if got := client.message.GetTo(); len(got) != 1 || got[0].Address != "to@example.com" {
		t.Fatalf("To = %v", got)
	}
	if got := client.message.GetGenHeader(mail.HeaderSubject); len(got) != 1 || got[0] != "Welcome" {
		t.Fatalf("Subject = %v", got)
	}
	var encoded bytes.Buffer
	if _, err := client.message.WriteTo(&encoded); err != nil {
		t.Fatalf("encode message: %v", err)
	}
	if !strings.Contains(encoded.String(), "Hello from SMTP.") {
		t.Fatalf("encoded message does not contain text body: %s", encoded.String())
	}
	for _, expected := range []string{"multipart/alternative", "text/plain", "text/html", "https://example.com"} {
		if !strings.Contains(encoded.String(), expected) {
			t.Fatalf("encoded message does not contain %q: %s", expected, encoded.String())
		}
	}
}

func TestSMTPMailerRejectsInvalidRecipient(t *testing.T) {
	provider := &SMTPMailer{client: &smtpSenderStub{}, fromEmail: "from@example.com"}
	if err := provider.Send(context.Background(), Message{To: "not an email"}); err == nil {
		t.Fatal("Send() error = nil, want invalid recipient error")
	}
}

func TestSMTPMailerWrapsProviderError(t *testing.T) {
	client := &smtpSenderStub{err: errors.New("connection refused")}
	provider := &SMTPMailer{client: client, fromEmail: "from@example.com"}
	err := provider.Send(context.Background(), Message{To: "to@example.com"})
	if err == nil || !errors.Is(err, client.err) {
		t.Fatalf("Send() error = %v, want wrapped provider error", err)
	}
}
