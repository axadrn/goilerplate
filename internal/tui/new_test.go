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
	model.payment = "polar"
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
		"--payment", "polar",
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

func TestProjectsExampleQuestionUsesExampleDataWording(t *testing.T) {
	model := newModel()
	model.setStep(stepExample)
	question, description := model.question()
	if question != "Include the Projects example?" || description != "Complete CRUD code. Run task seed later if you want example data." {
		t.Fatalf("question = %q, description = %q", question, description)
	}
}

func TestFreeSelectionSkipsPaidQuestions(t *testing.T) {
	model := newModel()
	model.setStep(stepEdition)
	model.selectCurrent()
	model.moveForward()

	if model.step != stepDestination {
		t.Fatalf("step = %d, want destination", model.step)
	}
	model.setStep(stepName)
	model.moveForward()
	if model.step != stepReview {
		t.Fatalf("step = %d, want review", model.step)
	}
	model.moveBack()
	if model.step != stepName {
		t.Fatalf("back step = %d, want name", model.step)
	}
}

func TestFreePaidSelectionRequestsPricingImmediately(t *testing.T) {
	selection := newModel(false)
	selection.moveCursor(1)

	updated, command := selection.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(model)
	if !result.openPricing || result.submitted || result.step != stepEdition || command == nil {
		t.Fatalf("pricing = %v, submitted = %v, step = %d, command = %v", result.openPricing, result.submitted, result.step, command)
	}
}

func TestPaidAccountContinuesIntoProjectSetup(t *testing.T) {
	selection := newModel(true)
	selection.moveCursor(1)

	updated, _ := selection.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(model)
	if result.openPricing || result.edition != "paid" || result.step != stepDestination {
		t.Fatalf("pricing = %v, edition = %q, step = %d", result.openPricing, result.edition, result.step)
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
	if !strings.Contains(view.Content, "existing CLI command generates the project") {
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

func TestLayoutDoesNotJumpBetweenSteps(t *testing.T) {
	for _, width := range []int{40, 80, 100} {
		selection := newModel(true)
		selection.width = width
		selection.height = 24
		selection.edition = "paid"

		var wantWidth, wantHeight int
		for index, current := range []step{stepEdition, stepDestination, stepFramework, stepReview} {
			selection.setStep(current)
			content := selection.View().Content
			gotWidth, gotHeight := lipgloss.Width(content), lipgloss.Height(content)
			if index == 0 {
				wantWidth, wantHeight = gotWidth, gotHeight
				continue
			}
			if gotWidth != wantWidth || gotHeight != wantHeight {
				t.Fatalf("width %d step %d size = %dx%d, want %dx%d", width, current, gotWidth, gotHeight, wantWidth, wantHeight)
			}
		}
	}
}

func TestEditionOptionsStayOnSingleLines(t *testing.T) {
	selection := newModel(false)
	selection.width = 80

	if height := lipgloss.Height(selection.questionView()); height != 5 {
		t.Fatalf("edition question height = %d, want 5 single-line rows", height)
	}
}

func TestHeaderIsCompactAndFooterStaysAtTerminalEdge(t *testing.T) {
	selection := newModel(false)
	selection.width = 80
	selection.height = 24

	content := selection.View().Content
	if !strings.Contains(content, "goilerplate") || !strings.Contains(content, " new  1/5") || strings.Contains(content, "━") {
		t.Fatalf("header = %q", strings.Split(content, "\n")[2])
	}
	lines := strings.Split(content, "\n")
	if len(lines) != selection.height || !strings.Contains(lines[len(lines)-1], "choose") {
		t.Fatalf("view has %d lines and last line %q", len(lines), lines[len(lines)-1])
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

func TestPaidCopySeparatesTheLicenseFromAppBilling(t *testing.T) {
	selection := newModel()
	selection.setStep(stepEdition)
	selection.moveCursor(1)
	if view := selection.questionView(); !strings.Contains(view, "Paid") || !strings.Contains(view, "Open pricing") {
		t.Fatalf("edition view = %q", view)
	}
	selection.setStep(stepPayment)
	title, _ := selection.question()
	if title != "Choose your app's billing provider" {
		t.Fatalf("payment title = %q", title)
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
