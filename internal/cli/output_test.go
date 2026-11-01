package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

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
