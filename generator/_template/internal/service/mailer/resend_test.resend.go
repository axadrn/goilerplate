package mailer

import (
	"context"
	"errors"
	"testing"

	"github.com/resend/resend-go/v2"
)

type resendSenderStub struct {
	request *resend.SendEmailRequest
	err     error
}

func (s *resendSenderStub) SendWithContext(_ context.Context, request *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	s.request = request
	return &resend.SendEmailResponse{}, s.err
}

func TestResendMailerSendsMessage(t *testing.T) {
	client := &resendSenderStub{}
	provider := &ResendMailer{client: client, fromEmail: "from@example.com"}

	err := provider.Send(context.Background(), Message{
		To:      "to@example.com",
		Subject: "Welcome",
		Text:    "Hello from the test.",
		HTML:    "<p>Hello from the test.</p>",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if client.request == nil {
		t.Fatal("Send() did not call Resend")
	}
	if client.request.From != "from@example.com" || len(client.request.To) != 1 || client.request.To[0] != "to@example.com" {
		t.Fatalf("Resend request addresses = %#v", client.request)
	}
	if client.request.Subject != "Welcome" || client.request.Text != "Hello from the test." || client.request.Html != "<p>Hello from the test.</p>" {
		t.Fatalf("Resend request content = %#v", client.request)
	}
}

func TestResendMailerWrapsProviderError(t *testing.T) {
	client := &resendSenderStub{err: errors.New("provider unavailable")}
	provider := &ResendMailer{client: client, fromEmail: "from@example.com"}

	err := provider.Send(context.Background(), Message{To: "to@example.com"})
	if err == nil || !errors.Is(err, client.err) {
		t.Fatalf("Send() error = %v, want wrapped provider error", err)
	}
}
