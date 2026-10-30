package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestArgumentsUseTheStableNewCommand(t *testing.T) {
	model := newModel()
	model.inputs[0].SetValue("./rocket")
	model.inputs[1].SetValue("github.com/acme/rocket")
	model.inputs[2].SetValue("Rocket")
	model.framework = "datastar"
	model.database = "postgres"
	model.payment = "polar"
	model.mail = "resend"
	model.workspaces = true
	model.storage = true
	model.content["docs"] = true

	want := []string{
		"--name", "Rocket",
		"--module", "github.com/acme/rocket",
		"--framework", "datastar",
		"--database", "postgres",
		"--payment", "polar",
		"--mail", "resend",
		"--workspaces",
		"--oauth", "google,github",
		"--storage",
		"--content", "docs",
		"./rocket",
	}
	if got := model.arguments(); !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
}

func TestWizardStartsAtTheDestination(t *testing.T) {
	model := newModel()
	if model.step != stepDestination || model.progress() != "1/13" {
		t.Fatalf("step = %d, progress = %q", model.step, model.progress())
	}
	model.moveBack()
	if model.step != stepDestination {
		t.Fatalf("back step = %d, want destination", model.step)
	}
}

func TestDefaultsIncludeBothOAuthProvidersAndStripe(t *testing.T) {
	model := newModel()
	arguments := strings.Join(model.arguments(), " ")
	if !strings.Contains(arguments, "--payment stripe") || !strings.Contains(arguments, "--oauth google,github") {
		t.Fatalf("arguments = %q", arguments)
	}
}

func TestFrontendSelectionBuildsTheFrameworkFlag(t *testing.T) {
	model := newModel()
	model.setStep(stepFramework)
	model.moveCursor(1)
	model.selectCurrent()

	if model.framework != "datastar" {
		t.Fatalf("framework = %q", model.framework)
	}
	if arguments := strings.Join(model.arguments(), " "); !strings.Contains(arguments, "--module example.com/my-app --framework datastar") {
		t.Fatalf("arguments = %q", arguments)
	}
	model.setStep(stepReview)
	if view := model.reviewView(model.contentWidth()); !strings.Contains(view, "Datastar") {
		t.Fatalf("review = %q", view)
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
			selection.workspaces = true
			selection.storage = true
			selection.content["blog"] = true
			selection.content["docs"] = true
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
	selection.workspaces = true
	selection.storage = true
	selection.content["blog"] = true
	selection.content["docs"] = true
	selection.setStep(stepContent)

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
		selection := newModel()
		selection.width = width
		selection.height = 24

		var wantWidth, wantHeight int
		for index, current := range []step{stepDestination, stepFramework, stepReview} {
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

func TestFrontendOptionsStayOnSingleLines(t *testing.T) {
	selection := newModel()
	selection.width = 80
	selection.setStep(stepFramework)

	if height := lipgloss.Height(selection.questionView()); height != 7 {
		t.Fatalf("frontend question height = %d, want 7 single-line rows", height)
	}
}

func TestHeaderIsCompactAndFooterStaysAtTerminalEdge(t *testing.T) {
	selection := newModel()
	selection.width = 80
	selection.height = 24

	content := selection.View().Content
	if !strings.Contains(content, "goilerplate") || !strings.Contains(content, " new  1/13") || strings.Contains(content, "━") {
		t.Fatalf("header = %q", strings.Split(content, "\n")[2])
	}
	lines := strings.Split(content, "\n")
	if len(lines) != selection.height || !strings.Contains(lines[len(lines)-1], "continue") {
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

func TestPaymentOffersStripeAndPolar(t *testing.T) {
	selection := newModel()
	selection.setStep(stepPayment)
	options := selection.options()
	if len(options) != 2 || options[0].value != "stripe" || options[1].value != "polar" || selection.cursor != 0 {
		t.Fatalf("payment options = %#v, cursor = %d", options, selection.cursor)
	}
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

func TestFrontendOffersSvelteKitAndHeadless(t *testing.T) {
	selection := newModel()
	selection.setStep(stepFramework)
	options := selection.options()
	values := make([]string, 0, len(options))
	for _, option := range options {
		values = append(values, option.value)
	}
	if !reflect.DeepEqual(values, []string{"htmx", "datastar", "svelte", "headless"}) {
		t.Fatalf("frontends = %v", values)
	}
	if options[2].label != "SvelteKit" || options[2].description != "SvelteKit app on the JSON API, served by the Go binary." {
		t.Fatalf("SvelteKit option = %#v", options[2])
	}
	if options[3].label != "Headless" || !strings.HasPrefix(options[3].description, "Go backend with a JSON API and no native frontend") {
		t.Fatalf("headless option = %#v", options[3])
	}
	view := selection.questionView()
	if !strings.Contains(view, "SvelteKit") || !strings.Contains(view, "Headless") {
		t.Fatalf("frontend view = %q", view)
	}
}

func TestJSONAPIStepFollowsHtmxAndDatastar(t *testing.T) {
	for _, framework := range []int{0, 1} {
		selection := newModel()
		selection.setStep(stepFramework)
		selection.moveCursor(framework)
		selection = press(t, selection, tea.KeyEnter)
		if selection.step != stepAPI {
			t.Fatalf("%s: step = %d, want JSON API", selection.framework, selection.step)
		}
		title, _ := selection.question()
		if title != "Include the JSON API?" {
			t.Fatalf("title = %q", title)
		}
		if selection.api || selection.cursor != 0 {
			t.Fatalf("JSON API default = %v, cursor = %d", selection.api, selection.cursor)
		}
		selection.moveCursor(1)
		selection = press(t, selection, tea.KeyEnter)
		if !selection.api || selection.step != stepMCP {
			t.Fatalf("%s: api = %v, step = %d, want MCP", selection.framework, selection.api, selection.step)
		}
		selection.moveBack()
		if selection.step != stepAPI {
			t.Fatalf("back step = %d, want JSON API", selection.step)
		}
	}
}

func TestSvelteKitAndHeadlessSkipTheJSONAPIAndContentSteps(t *testing.T) {
	want := []step{stepDestination, stepModule, stepName, stepFramework, stepMCP, stepDatabase, stepPayment, stepMail, stepWorkspaces, stepOAuth, stepStorage, stepReview}
	for _, test := range []struct {
		cursor    int
		framework string
	}{
		{cursor: 2, framework: "svelte"},
		{cursor: 3, framework: "headless"},
	} {
		t.Run(test.framework, func(t *testing.T) {
			selection := newModel()
			selection.setStep(stepFramework)
			selection.moveCursor(test.cursor)
			selection.selectCurrent()
			if selection.framework != test.framework || !selection.api {
				t.Fatalf("framework = %q, api = %v", selection.framework, selection.api)
			}

			selection.setStep(stepDestination)
			visited := []step{selection.step}
			for selection.step != stepReview {
				selection.moveForward()
				visited = append(visited, selection.step)
			}
			if !reflect.DeepEqual(visited, want) {
				t.Fatalf("visited = %v, want %v", visited, want)
			}
			for index := len(want) - 1; index > 0; index-- {
				if selection.step != want[index] {
					t.Fatalf("back step = %d, want %d", selection.step, want[index])
				}
				selection.moveBack()
			}
			got := selection.progress()
			if got != fmt.Sprintf("1/%d", len(want)) {
				t.Fatalf("progress = %q", got)
			}
		})
	}
}

func TestProgressCountsOnlyAskedQuestions(t *testing.T) {
	for _, test := range []struct {
		framework string
		api       bool
		want      string
	}{
		{framework: "htmx", want: "13/13"},
		{framework: "datastar", want: "13/13"},
		{framework: "datastar", api: true, want: "14/14"},
		{framework: "svelte", api: true, want: "12/12"},
		{framework: "headless", api: true, want: "12/12"},
	} {
		selection := newModel()
		selection.framework = test.framework
		selection.api = test.api
		selection.setStep(stepReview)
		got := selection.progress()
		if got != test.want {
			t.Fatalf("%s progress = %q, want %q", test.framework, got, test.want)
		}
	}
}

func TestArgumentsEmitHeadlessOrTheJSONAPI(t *testing.T) {
	modules := []string{"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github"}
	for _, test := range []struct {
		name      string
		framework string
		api       bool
		frontend  []string
		content   bool
	}{
		{name: "svelte drops content", framework: "svelte", api: true, frontend: []string{"--framework", "svelte"}},
		{name: "headless drops content", framework: "headless", api: true, frontend: []string{"--headless"}},
		{name: "htmx with the JSON API", framework: "htmx", api: true, frontend: []string{"--framework", "htmx", "--api"}, content: true},
		{name: "datastar without the JSON API", framework: "datastar", frontend: []string{"--framework", "datastar"}, content: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection := newModel()
			selection.inputs[0].SetValue("./acme")
			selection.inputs[1].SetValue("example.com/acme")
			selection.inputs[2].SetValue("Acme")
			selection.framework = test.framework
			selection.api = test.api
			selection.content["blog"] = true
			want := append([]string{"--name", "Acme", "--module", "example.com/acme"}, test.frontend...)
			want = append(want, modules...)
			if test.content {
				want = append(want, "--content", "blog")
			}
			want = append(want, "./acme")
			got := selection.arguments()
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("arguments = %#v, want %#v", got, want)
			}
		})
	}
}

func TestReviewShowsHeadlessAndTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		framework string
		api       bool
		width     int
		want      []string
		reject    []string
	}{
		{framework: "svelte", api: true, width: 100, want: []string{"SvelteKit  ·  JSON API  ·  SQLite", "Storage No"}, reject: []string{"Content"}},
		{framework: "svelte", api: true, width: 50, want: []string{"SvelteKit · JSON API · SQLite", "Storage No"}, reject: []string{"Content"}},
		{framework: "headless", api: true, width: 100, want: []string{"Headless  ·  JSON API  ·  SQLite", "Storage No"}, reject: []string{"Content"}},
		{framework: "htmx", api: true, width: 100, want: []string{"htmx  ·  JSON API  ·  SQLite", "Content Blog"}},
		{framework: "datastar", width: 100, want: []string{"Datastar  ·  SQLite", "Content Blog"}, reject: []string{"JSON API"}},
		{framework: "headless", api: true, width: 50, want: []string{"Headless · JSON API · SQLite", "Storage No"}, reject: []string{"Content"}},
	} {
		selection := newModel()
		selection.width = test.width
		selection.framework = test.framework
		selection.api = test.api
		selection.content["blog"] = true
		selection.setStep(stepReview)
		view := selection.reviewView(selection.contentWidth())
		for _, want := range test.want {
			if !strings.Contains(view, want) {
				t.Fatalf("%s width %d review = %q, want %q", test.framework, test.width, view, want)
			}
		}
		for _, reject := range test.reject {
			if strings.Contains(view, reject) {
				t.Fatalf("%s width %d review = %q, rejects %q", test.framework, test.width, view, reject)
			}
		}
	}
}

func TestMCPStepFollowsTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		framework int
		api       bool
	}{
		{framework: 0, api: true},
		{framework: 1, api: true},
		{framework: 2},
		{framework: 3},
	} {
		selection := newModel()
		selection.setStep(stepFramework)
		selection.moveCursor(test.framework)
		selection = press(t, selection, tea.KeyEnter)
		if test.api {
			selection.moveCursor(1)
			selection = press(t, selection, tea.KeyEnter)
		}
		if selection.step != stepMCP {
			t.Fatalf("%s: step = %d, want MCP", selection.framework, selection.step)
		}
		title, hint := selection.question()
		if title != "Include the MCP server?" || !strings.Contains(hint, "personal API token") {
			t.Fatalf("question = %q, %q", title, hint)
		}
		if selection.mcp || selection.cursor != 0 {
			t.Fatalf("MCP default = %v, cursor = %d", selection.mcp, selection.cursor)
		}
		selection.moveCursor(1)
		selection = press(t, selection, tea.KeyEnter)
		if !selection.mcp || selection.step != stepDatabase {
			t.Fatalf("%s: mcp = %v, step = %d, want database", selection.framework, selection.mcp, selection.step)
		}
		selection.moveBack()
		if selection.step != stepMCP || selection.cursor != 1 {
			t.Fatalf("back step = %d, cursor = %d, want MCP with Yes", selection.step, selection.cursor)
		}
	}
}

func TestMCPStepIsSkippedWithoutTheJSONAPI(t *testing.T) {
	selection := newModel()
	selection.setStep(stepAPI)
	selection = press(t, selection, tea.KeyEnter)
	if selection.api || selection.step != stepDatabase {
		t.Fatalf("api = %v, step = %d, want database", selection.api, selection.step)
	}
}

func TestTurningTheJSONAPIOffDropsTheMCPServer(t *testing.T) {
	modules := []string{"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme"}
	selection := newModel()
	selection.inputs[0].SetValue("./acme")
	selection.inputs[1].SetValue("example.com/acme")
	selection.inputs[2].SetValue("Acme")
	selection.api = true
	selection.mcp = true
	got := selection.arguments()
	want := append([]string{"--name", "Acme", "--module", "example.com/acme", "--framework", "htmx", "--api", "--mcp"}, modules...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
	selection.setStep(stepAPI)
	selection.moveCursor(-1)
	selection = press(t, selection, tea.KeyEnter)
	if selection.step != stepDatabase {
		t.Fatalf("step = %d, want database", selection.step)
	}
	got = selection.arguments()
	want = append([]string{"--name", "Acme", "--module", "example.com/acme", "--framework", "htmx"}, modules...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
}

func TestArgumentsAndReviewShowTheMCPServer(t *testing.T) {
	for _, test := range []struct {
		name      string
		framework string
		width     int
		arguments []string
		review    string
	}{
		{
			name: "datastar", framework: "datastar", width: 120,
			arguments: []string{
				"--name", "Acme", "--module", "example.com/acme", "--framework", "datastar", "--api", "--mcp",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
			review: "Datastar  ·  JSON API  ·  MCP  ·  SQLite",
		},
		{
			name: "headless narrow", framework: "headless", width: 50,
			arguments: []string{
				"--name", "Acme", "--module", "example.com/acme", "--headless", "--mcp",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
			review: "Headless · JSON API · MCP",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection := newModel()
			selection.inputs[0].SetValue("./acme")
			selection.inputs[1].SetValue("example.com/acme")
			selection.inputs[2].SetValue("Acme")
			selection.width = test.width
			selection.framework = test.framework
			selection.api = true
			selection.mcp = true
			got := selection.arguments()
			if !reflect.DeepEqual(got, test.arguments) {
				t.Fatalf("arguments = %#v, want %#v", got, test.arguments)
			}
			selection.setStep(stepReview)
			view := selection.reviewView(selection.contentWidth())
			if !strings.Contains(view, test.review) {
				t.Fatalf("review = %q, want %q", view, test.review)
			}
		})
	}
}

func press(t *testing.T, selection model, code rune) model {
	t.Helper()
	updated, _ := selection.Update(tea.KeyPressMsg{Code: code})
	return updated.(model)
}

func TestWizardTakesTypingOnTheFirstStep(t *testing.T) {
	wizard := newModel()
	if !wizard.inputs[0].Focused() {
		t.Fatal("the project directory input is not focused on start")
	}
	updated, _ := wizard.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	if got := updated.(model).inputs[0].Value(); got != "./my-appx" {
		t.Fatalf("directory after typing = %q, want ./my-appx", got)
	}
}
