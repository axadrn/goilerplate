package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/config"
)

func TestStyledWhoAmIUsesTheSharedTheme(t *testing.T) {
	output := &bytes.Buffer{}
	service := &fakeService{who: api.WhoAmIResponse{
		Account:  api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
		Licenses: []api.License{{ID: "paid", Status: api.LicenseStatusActive, Role: api.LicenseRoleMember}},
	}}
	app := testApp(output, &memoryStore{configuration: config.Config{SessionToken: "session"}}, &fakeDevice{}, service)
	app.StyledOutput = true

	if err := app.Run(context.Background(), []string{"whoami"}); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, value := range []string{"\x1b[", "goilerplate", "account", "@axadrn", "Email", "Access", "Paid", "Licenses", "active", "member"} {
		if !strings.Contains(got, value) {
			t.Fatalf("styled whoami does not contain %q: %q", value, got)
		}
	}
}

func TestStyledPricingUsesTheSharedTheme(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output, &memoryStore{}, &fakeDevice{}, &fakeService{})
	app.StyledOutput = true
	app.OpenBrowser = func(context.Context, string) error { return nil }

	app.openPricing(context.Background(), config.Config{})
	got := output.String()
	for _, value := range []string{"\x1b[", "goilerplate", "paid", "✓", "Pricing opened", "›", "No new login needed"} {
		if !strings.Contains(got, value) {
			t.Fatalf("styled pricing does not contain %q: %q", value, got)
		}
	}
}

func TestSharedThemeCoversEveryMessageKind(t *testing.T) {
	output := &bytes.Buffer{}
	app := &App{Out: output, StyledOutput: true}
	app.printHeading("test")
	app.printSuccess("done")
	app.printInfo("next")
	app.printWarning("careful")
	app.printField("Value", "content")
	app.printCommand("new", "Generate")
	app.printListItem("file.go")

	got := output.String()
	for _, value := range []string{"\x1b[", "✓", "›", "!", "Value", "new", "•"} {
		if !strings.Contains(got, value) {
			t.Fatalf("styled output does not contain %q: %q", value, got)
		}
	}
	if got := RenderError(errors.New("broken"), true); !strings.Contains(got, "\x1b[") || !strings.Contains(got, "✕") || !strings.Contains(got, "broken") {
		t.Fatalf("styled error = %q", got)
	}
}

func TestPlainOutputNeverAddsANSI(t *testing.T) {
	output := &bytes.Buffer{}
	app := &App{Out: output}
	app.printHeading("test")
	app.printSuccess("done")
	app.printInfo("next")
	app.printWarning("careful")
	app.printField("Value", "content")
	app.printCommand("new", "Generate")
	app.printListItem("file.go")

	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("plain output contains ANSI: %q", output.String())
	}
	if got := RenderError(errors.New("broken"), false); got != "Error: broken" {
		t.Fatalf("plain error = %q", got)
	}
}
