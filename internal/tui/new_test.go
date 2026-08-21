package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestFreeArgumentsUseTheStableNewCommand(t *testing.T) {
	model := newModel()
	model.inputs[0].SetValue("./acme")
	model.inputs[1].SetValue("example.com/acme")
	model.inputs[2].SetValue("Acme")

	want := []string{
		"--name", "Acme",
		"--module", "example.com/acme",
		"--edition", "free",
		"./acme",
	}
	if got := model.arguments(); !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
}

func TestPaidArgumentsUseTheStableNewCommand(t *testing.T) {
	model := newModel()
	model.inputs[0].SetValue("./rocket")
	model.inputs[1].SetValue("github.com/acme/rocket")
	model.inputs[2].SetValue("Rocket")
	model.edition = "paid"
	model.framework = "datastar"
	model.database = "postgres"
	model.payment = "lemon_squeezy"
	model.mail = "resend"
	model.teams = true
	model.storage = true
	model.content["docs"] = true
	model.example = true

	want := []string{
		"--name", "Rocket",
		"--module", "github.com/acme/rocket",
		"--edition", "paid",
		"--framework", "datastar",
		"--database", "postgres",
		"--payment", "lemon_squeezy",
		"--mail", "resend",
		"--teams",
		"--oauth", "google,github",
		"--storage",
		"--content", "docs",
		"--example",
		"./rocket",
	}
	if got := model.arguments(); !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
}

func TestFreeSelectionSkipsPaidQuestions(t *testing.T) {
	model := newModel()
	model.setStep(stepEdition)
	model.selectCurrent()
	model.moveForward()

	if model.step != stepReview {
		t.Fatalf("step = %d, want review", model.step)
	}
	model.moveBack()
	if model.step != stepEdition {
		t.Fatalf("back step = %d, want edition", model.step)
	}
}

func TestPaidDefaultsIncludeBothOAuthProviders(t *testing.T) {
	model := newModel()
	model.edition = "paid"
	arguments := strings.Join(model.arguments(), " ")
	if !strings.Contains(arguments, "--oauth google,github") {
		t.Fatalf("arguments = %q", arguments)
	}
}

func TestPaidFrontendSelectionBuildsTheFrameworkFlag(t *testing.T) {
	model := newModel()
	model.edition = "paid"
	model.setStep(stepFramework)
	model.moveCursor(1)
	model.selectCurrent()

	if model.framework != "datastar" {
		t.Fatalf("framework = %q", model.framework)
	}
	if arguments := strings.Join(model.arguments(), " "); !strings.Contains(arguments, "--framework datastar") {
		t.Fatalf("arguments = %q", arguments)
	}
}

func TestMultiSelectTogglesWithoutLeavingQuestion(t *testing.T) {
	selection := newModel()
	selection.setStep(stepOAuth)

	updated, _ := selection.Update(tea.KeyPressMsg{Text: " ", Code: ' '})
	result := updated.(model)
	if result.oauth["google"] || !result.oauth["github"] {
		t.Fatalf("OAuth selection = %#v", result.oauth)
	}
	if result.step != stepOAuth {
		t.Fatalf("step = %d, want OAuth", result.step)
	}
}

func TestEscapeOnFirstQuestionCancels(t *testing.T) {
	selection := newModel()
	updated, command := selection.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := updated.(model)
	if !result.cancelled || command == nil {
		t.Fatalf("cancelled = %v, command = %v", result.cancelled, command)
	}
}

func TestViewExplainsThatExistingCommandGenerates(t *testing.T) {
	model := newModel()
	model.setStep(stepReview)
	view := model.View()
	if !strings.Contains(view.Content, "existing CLI command") || !strings.Contains(view.Content, "performs the actual generation") {
		t.Fatalf("view = %q", view.Content)
	}
}

func TestReviewFitsStandardAndNarrowTerminals(t *testing.T) {
	for _, test := range []struct {
		name  string
		width int
	}{
		{name: "standard", width: 80},
		{name: "narrow", width: 40},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection := newModel()
			selection.width = test.width
			selection.height = 24
			selection.inputs[0].SetValue(strings.Repeat("d", 240))
			selection.inputs[1].SetValue(strings.Repeat("m", 240))
			selection.inputs[2].SetValue(strings.Repeat("n", 100))
			selection.edition = "paid"
			selection.teams = true
			selection.storage = true
			selection.content["blog"] = true
			selection.content["docs"] = true
			selection.example = true
			selection.setStep(stepReview)

			content := selection.View().Content
			if width := lipgloss.Width(content); width > test.width {
				t.Fatalf("view width = %d, terminal width = %d", width, test.width)
			}
			if height := lipgloss.Height(content); height > 24 {
				t.Fatalf("view height = %d, terminal height = 24", height)
			}
		})
	}
}

func TestWideSummaryFitsWithMaximumInputLengths(t *testing.T) {
	selection := newModel()
	selection.width = 100
	selection.height = 24
	selection.inputs[0].SetValue(strings.Repeat("d", 240))
	selection.inputs[1].SetValue(strings.Repeat("m", 240))
	selection.inputs[2].SetValue(strings.Repeat("n", 100))
	selection.edition = "paid"
	selection.teams = true
	selection.storage = true
	selection.content["blog"] = true
	selection.content["docs"] = true
	selection.example = true
	selection.setStep(stepExample)

	content := selection.View().Content
	if width := lipgloss.Width(content); width > 100 {
		t.Fatalf("view width = %d, terminal width = 100", width)
	}
	if height := lipgloss.Height(content); height > 24 {
		t.Fatalf("view height = %d, terminal height = 24", height)
	}
}

func TestBackgroundMessageSelectsReadableLightTheme(t *testing.T) {
	selection := newModel()
	darkBrand := selection.styles.brand.Render("goilerplate")

	updated, _ := selection.Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#FFFFFF")})
	light := updated.(model)
	lightBrand := light.styles.brand.Render("goilerplate")

	if darkBrand == lightBrand {
		t.Fatal("light background kept the dark-terminal accent palette")
	}
	if value := light.styles.value.Render("Value"); value != "Value" {
		t.Fatalf("value style overrides the terminal foreground: %q", value)
	}
}

func TestTinyTerminalShowsResizeMessage(t *testing.T) {
	selection := newModel()
	selection.width = 30
	selection.height = 12

	content := selection.View().Content
	if !strings.Contains(content, "Terminal too small") {
		t.Fatalf("view = %q", content)
	}
	if width := lipgloss.Width(content); width > 30 {
		t.Fatalf("view width = %d, terminal width = 30", width)
	}
	if height := lipgloss.Height(content); height > 12 {
		t.Fatalf("view height = %d, terminal height = 12", height)
	}
}
