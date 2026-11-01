package service

import (
	"errors"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/testutil"
)

func TestEmailServiceBuildsMagicLinkMessage(t *testing.T) {
	provider := testutil.NewMemoryMailer()
	service := NewEmailService(provider, "https://app.example.com", "Example App")

	if err := service.SendMagicLinkEmail("reader@example.com", "secret-token", "Reader"); err != nil {
		t.Fatalf("SendMagicLinkEmail() error = %v", err)
	}
	messages := provider.Messages()
	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	message := messages[0]
	if message.To != "reader@example.com" {
		t.Fatalf("recipient = %q", message.To)
	}
	if !strings.Contains(message.Text, "https://app.example.com/auth/magic-link/secret-token") {
		t.Fatalf("plain text message does not contain magic link: %q", message.Text)
	}
	if !strings.Contains(message.HTML, `href="https://app.example.com/auth/magic-link/secret-token"`) {
		t.Fatalf("HTML message does not contain clickable magic link: %q", message.HTML)
	}
	if !strings.Contains(message.HTML, ">Sign in</a>") {
		t.Fatalf("HTML message does not contain sign-in action: %q", message.HTML)
	}
}

func TestEmailRendererEscapesHTML(t *testing.T) {
	body, err := renderEmail(emailContent{
		Subject:     "Welcome",
		Heading:     `<script>alert("heading")</script>`,
		Intro:       []string{`Hello <strong>reader</strong>`},
		ActionLabel: "Open",
		ActionURL:   "https://example.com/path?one=1&two=2",
		AppName:     `<script>alert("app")</script>`,
	})
	if err != nil {
		t.Fatalf("renderEmail() error = %v", err)
	}
	if strings.Contains(body.HTML, "<script>") || strings.Contains(body.HTML, "<strong>") {
		t.Fatalf("rendered HTML contains unescaped input: %q", body.HTML)
	}
	for _, expected := range []string{"&lt;script&gt;", "&lt;strong&gt;", "one=1&amp;two=2"} {
		if !strings.Contains(body.HTML, expected) {
			t.Fatalf("rendered HTML does not contain %q: %q", expected, body.HTML)
		}
	}
}

func TestActionEmailTemplatesRenderClickableLinks(t *testing.T) {
	const actionURL = "https://app.example.com/action/secret-token"
	tests := []struct {
		name    string
		content emailContent
	}{
		{name: "Forgot password", content: forgotPasswordEmailTemplate(actionURL, "Example App")},
		{name: "Magic link", content: magicLinkEmailTemplate(actionURL, "Example App")},
		{name: "Welcome", content: welcomeEmailTemplate("Reader", actionURL, "Example App")},
		{name: "Workspace invitation", content: workspaceInvitationEmailTemplate(actionURL, "Example Team", "Owner", "Example App")},
		{name: "Email change verification", content: emailChangeVerificationTemplate("Reader", actionURL, "Example App")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := renderEmail(test.content)
			if err != nil {
				t.Fatalf("renderEmail() error = %v", err)
			}
			if !strings.Contains(body.Text, actionURL) {
				t.Fatalf("plain text message does not contain action URL: %q", body.Text)
			}
			if !strings.Contains(body.HTML, `href="`+actionURL+`"`) {
				t.Fatalf("HTML message does not contain clickable action URL: %q", body.HTML)
			}
		})
	}
}

func TestEmailServicePropagatesMailerError(t *testing.T) {
	providerError := errors.New("mailer unavailable")
	provider := testutil.NewMemoryMailer()
	provider.SetError(providerError)
	service := NewEmailService(provider, "https://app.example.com", "Example App")

	err := service.SendWelcomeEmail("reader@example.com", "Reader")
	if !errors.Is(err, providerError) {
		t.Fatalf("SendWelcomeEmail() error = %v, want provider error", err)
	}
}
