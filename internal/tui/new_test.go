package tui

import (
	"fmt"
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
		"--framework", "htmx",
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
	model.workspaces = true
	model.storage = true
	model.content["docs"] = true

	want := []string{
		"--name", "Rocket",
		"--module", "github.com/acme/rocket",
		"--edition", "paid",
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
	if model.step != stepFramework {
		t.Fatalf("step = %d, want framework", model.step)
	}
	model.moveForward()
	if model.step != stepAPI {
		t.Fatalf("step = %d, want JSON API", model.step)
	}
	model.moveForward()
	if model.step != stepReview {
		t.Fatalf("step = %d, want review", model.step)
	}
	model.moveBack()
	if model.step != stepAPI {
		t.Fatalf("back step = %d, want JSON API", model.step)
	}
	model.moveBack()
	if model.step != stepFramework {
		t.Fatalf("back step = %d, want framework", model.step)
	}
}

func TestFreeFrontendSelectionBuildsTheFrameworkFlag(t *testing.T) {
	model := newModel()
	model.setStep(stepFramework)
	model.moveCursor(1)
	model.selectCurrent()

	if model.framework != "datastar" {
		t.Fatalf("framework = %q", model.framework)
	}
	if arguments := strings.Join(model.arguments(), " "); !strings.Contains(arguments, "--edition free --framework datastar") {
		t.Fatalf("arguments = %q", arguments)
	}
	model.setStep(stepReview)
	if view := model.reviewView(model.contentWidth()); !strings.Contains(view, "Datastar") {
		t.Fatalf("review = %q", view)
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
	selection.edition = "paid"
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
	if !strings.Contains(content, "goilerplate") || !strings.Contains(content, " new  1/7") || strings.Contains(content, "━") {
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

func TestFrontendOffersSvelteKitAndHeadlessInBothEditions(t *testing.T) {
	for _, edition := range []string{"free", "paid"} {
		t.Run(edition, func(t *testing.T) {
			selection := newModel(true)
			selection.edition = edition
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
		})
	}
}

func TestJSONAPIStepFollowsHtmxAndDatastar(t *testing.T) {
	for _, test := range []struct {
		edition   string
		framework int
		next      step
	}{
		{edition: "free", framework: 0, next: stepMCP},
		{edition: "free", framework: 1, next: stepMCP},
		{edition: "paid", framework: 0, next: stepMCP},
		{edition: "paid", framework: 1, next: stepMCP},
	} {
		selection := newModel(true)
		selection.edition = test.edition
		selection.setStep(stepFramework)
		selection.moveCursor(test.framework)
		selection = press(t, selection, tea.KeyEnter)
		if selection.step != stepAPI {
			t.Fatalf("%s %s: step = %d, want JSON API", test.edition, selection.framework, selection.step)
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
		if !selection.api || selection.step != test.next {
			t.Fatalf("%s %s: api = %v, step = %d, want %d", test.edition, selection.framework, selection.api, selection.step, test.next)
		}
		selection.moveBack()
		if selection.step != stepAPI {
			t.Fatalf("back step = %d, want JSON API", selection.step)
		}
	}
}

func TestSvelteKitAndHeadlessSkipTheJSONAPIAndContentSteps(t *testing.T) {
	for _, test := range []struct {
		edition   string
		cursor    int
		framework string
		want      []step
	}{
		{edition: "free", cursor: 2, framework: "svelte", want: []step{stepEdition, stepDestination, stepModule, stepName, stepFramework, stepMCP, stepReview}},
		{edition: "free", cursor: 3, framework: "headless", want: []step{stepEdition, stepDestination, stepModule, stepName, stepFramework, stepMCP, stepReview}},
		{edition: "paid", cursor: 2, framework: "svelte", want: []step{stepEdition, stepDestination, stepModule, stepName, stepFramework, stepMCP, stepDatabase, stepPayment, stepMail, stepWorkspaces, stepOAuth, stepStorage, stepReview}},
		{edition: "paid", cursor: 3, framework: "headless", want: []step{stepEdition, stepDestination, stepModule, stepName, stepFramework, stepMCP, stepDatabase, stepPayment, stepMail, stepWorkspaces, stepOAuth, stepStorage, stepReview}},
	} {
		t.Run(test.edition+" "+test.framework, func(t *testing.T) {
			selection := newModel(true)
			selection.edition = test.edition
			selection.setStep(stepFramework)
			selection.moveCursor(test.cursor)
			selection.selectCurrent()
			if selection.framework != test.framework || !selection.api {
				t.Fatalf("framework = %q, api = %v", selection.framework, selection.api)
			}

			selection.setStep(stepEdition)
			visited := []step{selection.step}
			for selection.step != stepReview {
				selection.moveForward()
				visited = append(visited, selection.step)
			}
			if !reflect.DeepEqual(visited, test.want) {
				t.Fatalf("visited = %v, want %v", visited, test.want)
			}
			for index := len(test.want) - 1; index > 0; index-- {
				if selection.step != test.want[index] {
					t.Fatalf("back step = %d, want %d", selection.step, test.want[index])
				}
				selection.moveBack()
			}
			got := selection.progress()
			if got != fmt.Sprintf("1/%d", len(test.want)) {
				t.Fatalf("progress = %q", got)
			}
		})
	}
}

func TestProgressCountsOnlyAskedQuestions(t *testing.T) {
	for _, test := range []struct {
		edition   string
		framework string
		want      string
	}{
		{edition: "free", framework: "htmx", want: "7/7"},
		{edition: "free", framework: "headless", want: "7/7"},
		{edition: "free", framework: "svelte", want: "7/7"},
		{edition: "paid", framework: "svelte", want: "13/13"},
		{edition: "paid", framework: "datastar", want: "14/14"},
		{edition: "paid", framework: "headless", want: "13/13"},
	} {
		selection := newModel(true)
		selection.edition = test.edition
		selection.framework = test.framework
		selection.setStep(stepReview)
		got := selection.progress()
		if got != test.want {
			t.Fatalf("%s %s progress = %q, want %q", test.edition, test.framework, got, test.want)
		}
	}
}

func TestArgumentsEmitHeadlessOrTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		name      string
		edition   string
		framework string
		api       bool
		want      []string
	}{
		{
			name: "free headless", edition: "free", framework: "headless", api: true,
			want: []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--headless", "./acme"},
		},
		{
			name: "free svelte", edition: "free", framework: "svelte", api: true,
			want: []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--framework", "svelte", "./acme"},
		},
		{
			name: "paid svelte drops content", edition: "paid", framework: "svelte", api: true,
			want: []string{
				"--name", "Acme", "--module", "example.com/acme", "--edition", "paid", "--framework", "svelte",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
		},
		{
			name: "free htmx with the JSON API", edition: "free", framework: "htmx", api: true,
			want: []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--framework", "htmx", "--api", "./acme"},
		},
		{
			name: "free datastar without the JSON API", edition: "free", framework: "datastar",
			want: []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--framework", "datastar", "./acme"},
		},
		{
			name: "paid headless drops content", edition: "paid", framework: "headless", api: true,
			want: []string{
				"--name", "Acme", "--module", "example.com/acme", "--edition", "paid", "--headless",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
		},
		{
			name: "paid datastar with the JSON API", edition: "paid", framework: "datastar", api: true,
			want: []string{
				"--name", "Acme", "--module", "example.com/acme", "--edition", "paid", "--framework", "datastar", "--api",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "--content", "blog", "./acme",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection := newModel(true)
			selection.inputs[0].SetValue("./acme")
			selection.inputs[1].SetValue("example.com/acme")
			selection.inputs[2].SetValue("Acme")
			selection.edition = test.edition
			selection.framework = test.framework
			selection.api = test.api
			selection.content["blog"] = true
			got := selection.arguments()
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("arguments = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestReviewShowsHeadlessAndTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		edition   string
		framework string
		api       bool
		width     int
		want      []string
		reject    []string
	}{
		{edition: "free", framework: "headless", api: true, width: 80, want: []string{"Free  ·  SQLite  ·  SMTP  ·  Headless  ·  JSON API"}},
		{edition: "free", framework: "htmx", api: true, width: 80, want: []string{"Free  ·  SQLite  ·  SMTP  ·  htmx  ·  JSON API"}},
		{edition: "free", framework: "svelte", api: true, width: 80, want: []string{"Free  ·  SQLite  ·  SMTP  ·  SvelteKit  ·  JSON API"}},
		{edition: "paid", framework: "svelte", api: true, width: 100, want: []string{"Paid  ·  SvelteKit  ·  JSON API  ·  SQLite", "Storage No"}, reject: []string{"Content"}},
		{edition: "paid", framework: "svelte", api: true, width: 50, want: []string{"Paid · SvelteKit · JSON API", "Storage No"}, reject: []string{"Content"}},
		{edition: "free", framework: "datastar", width: 80, want: []string{"Free  ·  SQLite  ·  SMTP  ·  Datastar"}, reject: []string{"JSON API"}},
		{edition: "free", framework: "headless", api: true, width: 50, want: []string{"Free · SQLite · SMTP · Headless · JSON API"}},
		{edition: "paid", framework: "headless", api: true, width: 100, want: []string{"Paid  ·  Headless  ·  JSON API  ·  SQLite", "Storage No"}, reject: []string{"Content"}},
		{edition: "paid", framework: "htmx", api: true, width: 100, want: []string{"Paid  ·  htmx  ·  JSON API  ·  SQLite", "Content Blog"}},
		{edition: "paid", framework: "headless", api: true, width: 50, want: []string{"Paid · Headless · JSON API", "Storage No"}, reject: []string{"Content"}},
	} {
		selection := newModel(true)
		selection.width = test.width
		selection.edition = test.edition
		selection.framework = test.framework
		selection.api = test.api
		selection.content["blog"] = true
		selection.setStep(stepReview)
		view := selection.reviewView(selection.contentWidth())
		for _, want := range test.want {
			if !strings.Contains(view, want) {
				t.Fatalf("%s %s width %d review = %q, want %q", test.edition, test.framework, test.width, view, want)
			}
		}
		for _, reject := range test.reject {
			if strings.Contains(view, reject) {
				t.Fatalf("%s %s width %d review = %q, rejects %q", test.edition, test.framework, test.width, view, reject)
			}
		}
	}
}

func TestMCPStepFollowsTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		edition   string
		framework int
		api       bool
		next      step
	}{
		{edition: "free", framework: 0, api: true, next: stepReview},
		{edition: "free", framework: 2, next: stepReview},
		{edition: "paid", framework: 1, api: true, next: stepDatabase},
		{edition: "paid", framework: 2, next: stepDatabase},
	} {
		selection := newModel(true)
		selection.edition = test.edition
		selection.setStep(stepFramework)
		selection.moveCursor(test.framework)
		selection = press(t, selection, tea.KeyEnter)
		if test.api {
			selection.moveCursor(1)
			selection = press(t, selection, tea.KeyEnter)
		}
		if selection.step != stepMCP {
			t.Fatalf("%s %s: step = %d, want MCP", test.edition, selection.framework, selection.step)
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
		if !selection.mcp || selection.step != test.next {
			t.Fatalf("%s %s: mcp = %v, step = %d, want %d", test.edition, selection.framework, selection.mcp, selection.step, test.next)
		}
		selection.moveBack()
		if selection.step != stepMCP || selection.cursor != 1 {
			t.Fatalf("back step = %d, cursor = %d, want MCP with Yes", selection.step, selection.cursor)
		}
	}
}

func TestMCPStepIsSkippedWithoutTheJSONAPI(t *testing.T) {
	for _, edition := range []string{"free", "paid"} {
		selection := newModel(true)
		selection.edition = edition
		selection.setStep(stepAPI)
		selection = press(t, selection, tea.KeyEnter)
		if selection.api || selection.step == stepMCP {
			t.Fatalf("%s: api = %v, step = %d", edition, selection.api, selection.step)
		}
	}
}

func TestTurningTheJSONAPIOffDropsTheMCPServer(t *testing.T) {
	selection := newModel(true)
	selection.inputs[0].SetValue("./acme")
	selection.inputs[1].SetValue("example.com/acme")
	selection.inputs[2].SetValue("Acme")
	selection.api = true
	selection.mcp = true
	got := selection.arguments()
	want := []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--framework", "htmx", "--api", "--mcp", "./acme"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
	selection.setStep(stepAPI)
	selection.moveCursor(-1)
	selection = press(t, selection, tea.KeyEnter)
	if selection.step != stepReview {
		t.Fatalf("step = %d, want review", selection.step)
	}
	got = selection.arguments()
	want = []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--framework", "htmx", "./acme"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v, want %#v", got, want)
	}
}

func TestArgumentsAndReviewShowTheMCPServer(t *testing.T) {
	for _, test := range []struct {
		name      string
		edition   string
		framework string
		api       bool
		width     int
		arguments []string
		review    string
	}{
		{
			name: "free headless", edition: "free", framework: "headless", api: true, width: 80,
			arguments: []string{"--name", "Acme", "--module", "example.com/acme", "--edition", "free", "--headless", "--mcp", "./acme"},
			review:    "Free  ·  SQLite  ·  SMTP  ·  Headless  ·  JSON API  ·  MCP",
		},
		{
			name: "paid datastar", edition: "paid", framework: "datastar", api: true, width: 120,
			arguments: []string{
				"--name", "Acme", "--module", "example.com/acme", "--edition", "paid", "--framework", "datastar", "--api", "--mcp",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
			review: "Paid  ·  Datastar  ·  JSON API  ·  MCP  ·  SQLite",
		},
		{
			name: "paid headless narrow", edition: "paid", framework: "headless", api: true, width: 50,
			arguments: []string{
				"--name", "Acme", "--module", "example.com/acme", "--edition", "paid", "--headless", "--mcp",
				"--database", "sqlite", "--payment", "stripe", "--mail", "smtp", "--oauth", "google,github", "./acme",
			},
			review: "Paid · Headless · JSON API · MCP",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection := newModel(true)
			selection.inputs[0].SetValue("./acme")
			selection.inputs[1].SetValue("example.com/acme")
			selection.inputs[2].SetValue("Acme")
			selection.width = test.width
			selection.edition = test.edition
			selection.framework = test.framework
			selection.api = test.api
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
