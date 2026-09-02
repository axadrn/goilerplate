package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var ErrCancelled = errors.New("project setup cancelled")

type step int

const (
	stepEdition step = iota
	stepDestination
	stepModule
	stepName
	stepFramework
	stepDatabase
	stepPayment
	stepMail
	stepWorkspaces
	stepOAuth
	stepStorage
	stepContent
	stepDemo
	stepReview
)

type option struct {
	label       string
	description string
	value       string
}

type styles struct {
	title   lipgloss.Style
	brand   lipgloss.Style
	label   lipgloss.Style
	muted   lipgloss.Style
	value   lipgloss.Style
	success lipgloss.Style
	active  lipgloss.Style
}

type Result struct {
	Arguments   []string
	OpenPricing bool
}

type model struct {
	step        step
	inputs      [3]textinput.Model
	cursor      int
	width       int
	height      int
	edition     string
	framework   string
	database    string
	payment     string
	mail        string
	workspaces  bool
	oauth       map[string]bool
	storage     bool
	content     map[string]bool
	demo        bool
	paidAccess  bool
	submitted   bool
	openPricing bool
	cancelled   bool
	styles      styles
}

func newStyles(isDark bool) styles {
	color := lipgloss.LightDark(isDark)
	muted := color(lipgloss.Color("#52525B"), lipgloss.Color("#A1A1AA"))
	purple := color(lipgloss.Color("#6D28D9"), lipgloss.Color("#A78BFA"))
	fuchsia := color(lipgloss.Color("#A21CAF"), lipgloss.Color("#F0ABFC"))
	green := color(lipgloss.Color("#047857"), lipgloss.Color("#34D399"))
	activeText := color(lipgloss.Color("#701A75"), lipgloss.Color("#FAFAFA"))
	activeBackground := color(lipgloss.Color("#FAE8FF"), lipgloss.Color("#701A75"))

	return styles{
		title:   lipgloss.NewStyle().Bold(true),
		brand:   lipgloss.NewStyle().Bold(true).Foreground(purple),
		label:   lipgloss.NewStyle().Bold(true).Foreground(fuchsia),
		muted:   lipgloss.NewStyle().Foreground(muted),
		value:   lipgloss.NewStyle(),
		success: lipgloss.NewStyle().Foreground(green),
		active: lipgloss.NewStyle().
			Foreground(activeText).
			Background(activeBackground).
			Padding(0, 1),
	}
}

func Run(ctx context.Context, input io.Reader, output io.Writer, paidAccess bool) (Result, error) {
	program := tea.NewProgram(
		newModel(paidAccess),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, context.Canceled) {
			return Result{}, ErrCancelled
		}
		return Result{}, fmt.Errorf("run project setup: %w", err)
	}
	result, ok := final.(model)
	if !ok || result.cancelled || !result.submitted && !result.openPricing {
		return Result{}, ErrCancelled
	}
	return Result{Arguments: result.arguments(), OpenPricing: result.openPricing}, nil
}

func newModel(paidAccess ...bool) model {
	destination := textinput.New()
	destination.Prompt = "› "
	destination.Placeholder = "./my-app"
	destination.SetValue("./my-app")
	destination.CharLimit = 240
	destination.SetWidth(48)

	module := textinput.New()
	module.Prompt = "› "
	module.Placeholder = "example.com/my-app"
	module.SetValue("example.com/my-app")
	module.CharLimit = 240
	module.SetWidth(48)

	name := textinput.New()
	name.Prompt = "› "
	name.Placeholder = "My App"
	name.SetValue("My App")
	name.CharLimit = 100
	name.SetWidth(48)

	return model{
		inputs:     [3]textinput.Model{destination, module, name},
		width:      80,
		height:     24,
		edition:    "free",
		framework:  "htmx",
		database:   "sqlite",
		payment:    "stripe",
		mail:       "smtp",
		oauth:      map[string]bool{"google": true, "github": true},
		content:    map[string]bool{},
		paidAccess: len(paidAccess) > 0 && paidAccess[0],
		styles:     newStyles(true),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.RequestBackgroundColor)
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.BackgroundColorMsg:
		isDark := message.IsDark()
		m.styles = newStyles(isDark)
		for index := range m.inputs {
			m.inputs[index].SetStyles(textinput.DefaultStyles(isDark))
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		inputWidth := min(52, max(20, message.Width-12))
		for index := range m.inputs {
			m.inputs[index].SetWidth(inputWidth)
		}
		return m, nil
	case tea.KeyPressMsg:
		key := message.String()
		if key == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.isTextStep() {
			switch key {
			case "enter":
				if strings.TrimSpace(m.currentInput().Value()) != "" {
					m.moveForward()
				}
				return m, nil
			case "esc":
				m.moveBack()
				return m, nil
			}
			index := m.inputIndex()
			updated, command := m.inputs[index].Update(message)
			m.inputs[index] = updated
			return m, command
		}

		switch key {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc", "left", "h":
			if m.step == stepEdition {
				m.cancelled = true
				return m, tea.Quit
			}
			m.moveBack()
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		case " ", "space":
			if m.isMultiStep() {
				m.toggleCurrent()
			}
			return m, nil
		case "enter", "right", "l":
			if m.step == stepReview {
				m.submitted = true
				return m, tea.Quit
			}
			if m.isMultiStep() {
				m.moveForward()
				return m, nil
			}
			m.selectCurrent()
			if m.step == stepEdition && m.edition == "paid" && !m.paidAccess {
				m.openPricing = true
				return m, tea.Quit
			}
			m.moveForward()
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.width < 40 || m.height < 18 {
		view := tea.NewView(strings.Join([]string{
			m.styles.brand.Render("goilerplate"),
			"",
			"Terminal too small.",
			"Resize to at least 40 × 18.",
			"",
			m.styles.muted.Render("ctrl+c quit"),
		}, "\n"))
		view.AltScreen = true
		view.WindowTitle = "goilerplate new"
		return view
	}
	leftMargin := 4
	if m.width < 60 {
		leftMargin = 2
	}
	topMargin := 2
	if m.height < 22 {
		topMargin = 1
	}
	contentWidth := m.width - leftMargin - 2
	header := m.styles.brand.Render("goilerplate") + m.styles.muted.Render(" new  "+m.progress())
	main := lipgloss.NewStyle().Width(contentWidth).Render(strings.Join([]string{
		header,
		"",
		m.questionView(),
	}, "\n"))
	lines := strings.Split(main, "\n")
	blankRows := max(1, m.height-topMargin-len(lines)-1)
	lines = append(lines, make([]string, blankRows)...)
	lines = append(lines, m.footerView())
	content := strings.Join(lines, "\n")
	view := tea.NewView(lipgloss.NewStyle().MarginLeft(leftMargin).MarginTop(topMargin).Render(content))
	view.AltScreen = true
	view.WindowTitle = "goilerplate new"
	return view
}

func (m model) questionView() string {
	title, hint := m.question()
	parts := []string{m.styles.title.Render(title)}
	lineWidth := m.contentWidth()
	if m.width >= 60 && hint != "" {
		parts = append(parts, m.styles.muted.Render(shorten(hint, lineWidth)))
	}
	parts = append(parts, "")
	if m.isTextStep() {
		parts = append(parts, m.currentInput().View())
		return strings.Join(parts, "\n")
	}
	if m.step == stepReview {
		parts = append(parts, m.reviewView(lineWidth), "", m.styles.active.Render("Generate project"))
		return strings.Join(parts, "\n")
	}
	options := m.options()
	labelWidth := 0
	for _, option := range options {
		labelWidth = max(labelWidth, lipgloss.Width(option.label))
	}
	for index, option := range options {
		cursor := "  "
		if index == m.cursor {
			cursor = m.styles.label.Render("› ")
		}
		marker := "○"
		if m.optionSelected(option.value) {
			marker = m.styles.success.Render("●")
		}
		if m.isMultiStep() {
			marker = "[ ]"
			if m.optionSelected(option.value) {
				marker = m.styles.success.Render("[x]")
			}
		}
		plainLabel := option.label + strings.Repeat(" ", labelWidth-lipgloss.Width(option.label))
		label := m.styles.value.Render(plainLabel)
		if index == m.cursor {
			label = m.styles.label.Render(plainLabel)
		}
		row := fmt.Sprintf("%s%s  %s", cursor, marker, label)
		if option.description != "" {
			remaining := lineWidth - 7 - lipgloss.Width(marker) - labelWidth
			if remaining >= 12 {
				row += "   " + m.styles.muted.Render(shorten(option.description, remaining))
			}
		}
		parts = append(parts, row)
	}
	return strings.Join(parts, "\n")
}

func (m model) reviewView(lineWidth int) string {
	if m.width < 60 {
		rows := []string{
			m.styles.label.Render(shorten(strings.TrimSpace(m.inputs[2].Value()), lineWidth)),
			m.styles.muted.Render(shorten("Module  "+strings.TrimSpace(m.inputs[1].Value()), lineWidth)),
			m.styles.muted.Render(shorten("Folder  "+strings.TrimSpace(m.inputs[0].Value()), lineWidth)),
		}
		if m.edition == "free" {
			return strings.Join(append(rows, m.styles.value.Render(shorten("Free · SQLite · SMTP · htmx", lineWidth))), "\n")
		}
		return strings.Join(append(rows,
			m.styles.value.Render(shorten("Paid · "+displayValue(m.framework)+" · "+displayValue(m.database), lineWidth)),
			m.styles.value.Render(shorten(displayValue(m.payment)+" · "+displayValue(m.mail), lineWidth)),
			m.styles.value.Render(shorten(fmt.Sprintf("Workspaces %s · OAuth %d", yesNo(m.workspaces), len(selectedKeys(m.oauth))), lineWidth)),
			m.styles.value.Render(shorten(fmt.Sprintf("Storage %s · Content %d", yesNo(m.storage), len(selectedKeys(m.content))), lineWidth)),
		), "\n")
	}
	rows := []string{
		m.styles.label.Render(shorten(strings.TrimSpace(m.inputs[2].Value()), lineWidth)),
		m.styles.muted.Render(shorten("Module  "+strings.TrimSpace(m.inputs[1].Value()), lineWidth)),
		m.styles.muted.Render(shorten("Folder  "+strings.TrimSpace(m.inputs[0].Value()), lineWidth)),
	}
	if m.edition == "free" {
		return strings.Join(append(rows,
			m.styles.value.Render(shorten("Free  ·  SQLite  ·  SMTP  ·  htmx", lineWidth)),
		), "\n")
	}
	return strings.Join(append(rows,
		m.styles.value.Render(shorten("Paid  ·  "+displayValue(m.framework)+"  ·  "+displayValue(m.database)+"  ·  "+displayValue(m.payment)+"  ·  "+displayValue(m.mail), lineWidth)),
		m.styles.value.Render(shorten("Workspaces "+yesNo(m.workspaces)+"  ·  OAuth "+selectedValues(m.oauth), lineWidth)),
		m.styles.value.Render(shorten("Storage "+yesNo(m.storage)+"  ·  Content "+selectedValues(m.content)+"  ·  Demo "+yesNo(m.demo), lineWidth)),
	), "\n")
}

func (m model) footerView() string {
	if m.width < 60 {
		if m.isTextStep() {
			return m.styles.muted.Render("enter next · esc back · ctrl+c quit")
		}
		if m.step == stepReview {
			return m.styles.muted.Render("enter generate · esc back · q quit")
		}
		if m.isMultiStep() {
			return m.styles.muted.Render("↑↓ move · space toggle · enter next")
		}
		return m.styles.muted.Render("↑↓ choose · enter next · esc back")
	}
	if m.isTextStep() {
		return m.styles.muted.Render("enter continue   esc back   ctrl+c quit")
	}
	if m.step == stepReview {
		return m.styles.muted.Render("enter generate   esc back   q quit")
	}
	if m.isMultiStep() {
		return m.styles.muted.Render("↑↓ move   space toggle   enter continue   esc back")
	}
	return m.styles.muted.Render("↑↓ choose   enter continue   esc back   q quit")
}

func (m model) question() (string, string) {
	switch m.step {
	case stepDestination:
		return "Where should the project live?", "A new folder. It must be empty or not exist yet."
	case stepModule:
		return "What is the Go module path?", "Usually your repository path, for example github.com/acme/rocket."
	case stepName:
		return "What should the app be called?", "This is the human-readable product name."
	case stepEdition:
		return "Choose your edition", "Free is the complete foundation. Paid unlocks every product module and lifetime updates."
	case stepFramework:
		return "Choose a frontend", "htmx is the default. Datastar uses server-sent events and fine-grained patches."
	case stepDatabase:
		return "Choose a database", "SQLite is simple. PostgreSQL is ready for distributed deployments."
	case stepPayment:
		return "Choose your app's billing provider", "Only the selected provider is included in your project."
	case stepMail:
		return "Choose transactional email", "SMTP works everywhere. Resend offers a hosted API."
	case stepWorkspaces:
		return "Include Workspaces?", "Adds shared workspaces, roles, invitations, seats, and workspace billing."
	case stepOAuth:
		return "Include OAuth providers", "Choose any combination. Press Enter when ready."
	case stepStorage:
		return "Include file storage?", "Adds the S3-compatible storage package and configuration."
	case stepContent:
		return "Include content modules", "Blog and Docs share the same Markdown engine."
	case stepDemo:
		return "Generate the Demo?", "Fixed reference app with Projects data. Run task seed after generation."
	default:
		return "Ready to build", "The existing CLI command generates the project."
	}
}

func (m model) options() []option {
	switch m.step {
	case stepEdition:
		paid := option{label: "Paid", value: "paid", description: "Every module and lifetime updates."}
		if !m.paidAccess {
			paid.description = "Open pricing. Buy once, then run goilerplate new again."
		}
		return []option{
			{label: "Free", value: "free", description: "Complete foundation with SQLite, SMTP, htmx, auth, and security."},
			paid,
		}
	case stepFramework:
		return []option{
			{label: "htmx", value: "htmx", description: "Small, stable, and the default goilerplate frontend."},
			{label: "Datastar", value: "datastar", description: "Server-sent events with a reactive HTML-first client."},
		}
	case stepDatabase:
		return []option{{label: "SQLite", value: "sqlite"}, {label: "PostgreSQL", value: "postgres"}}
	case stepPayment:
		return []option{{label: "Stripe", value: "stripe"}, {label: "Polar", value: "polar"}}
	case stepMail:
		return []option{{label: "SMTP", value: "smtp"}, {label: "Resend", value: "resend"}}
	case stepWorkspaces, stepStorage, stepDemo:
		return []option{{label: "No", value: "false"}, {label: "Yes", value: "true"}}
	case stepOAuth:
		return []option{{label: "Google", value: "google"}, {label: "GitHub", value: "github"}}
	case stepContent:
		return []option{{label: "Blog", value: "blog"}, {label: "Docs", value: "docs"}}
	default:
		return nil
	}
}

func (m model) arguments() []string {
	arguments := []string{
		"--name", strings.TrimSpace(m.inputs[2].Value()),
		"--module", strings.TrimSpace(m.inputs[1].Value()),
		"--edition", m.edition,
	}
	if m.edition == "paid" {
		arguments = append(arguments,
			"--framework", m.framework,
			"--database", m.database,
			"--payment", m.payment,
			"--mail", m.mail,
		)
		if m.workspaces {
			arguments = append(arguments, "--workspaces")
		}
		if values := selectedKeys(m.oauth); len(values) > 0 {
			arguments = append(arguments, "--oauth", strings.Join(values, ","))
		}
		if m.storage {
			arguments = append(arguments, "--storage")
		}
		if values := selectedKeys(m.content); len(values) > 0 {
			arguments = append(arguments, "--content", strings.Join(values, ","))
		}
		if m.demo {
			arguments = append(arguments, "--demo")
		}
	}
	return append(arguments, strings.TrimSpace(m.inputs[0].Value()))
}

func (m model) isTextStep() bool {
	return m.step >= stepDestination && m.step <= stepName
}

func (m model) isMultiStep() bool {
	return m.step == stepOAuth || m.step == stepContent
}

func (m *model) currentInput() *textinput.Model {
	return &m.inputs[m.inputIndex()]
}

func (m model) inputIndex() int {
	return int(m.step - stepDestination)
}

func (m *model) moveForward() {
	if m.isTextStep() {
		m.currentInput().Blur()
	}
	switch {
	case m.step == stepEdition:
		m.setStep(stepDestination)
		return
	case m.step == stepName && m.edition == "free":
		m.setStep(stepReview)
		return
	case m.step == stepName:
		m.setStep(stepFramework)
		return
	}
	if m.step < stepReview {
		m.setStep(m.step + 1)
	}
}

func (m *model) moveBack() {
	if m.step == stepEdition {
		return
	}
	if m.step == stepDestination {
		m.currentInput().Blur()
		m.setStep(stepEdition)
		return
	}
	if m.step == stepReview && m.edition == "free" {
		m.setStep(stepName)
		return
	}
	if m.step == stepFramework {
		m.setStep(stepName)
		return
	}
	if m.isTextStep() {
		m.currentInput().Blur()
	}
	m.setStep(m.step - 1)
}

func (m *model) setStep(next step) {
	m.step = next
	m.cursor = m.selectedCursor()
	if m.isTextStep() {
		m.currentInput().Focus()
		m.currentInput().CursorEnd()
	}
}

func (m *model) moveCursor(change int) {
	options := m.options()
	if len(options) == 0 {
		return
	}
	m.cursor = (m.cursor + change + len(options)) % len(options)
}

func (m *model) selectCurrent() {
	options := m.options()
	if m.cursor < 0 || m.cursor >= len(options) {
		return
	}
	value := options[m.cursor].value
	switch m.step {
	case stepEdition:
		m.edition = value
	case stepFramework:
		m.framework = value
	case stepDatabase:
		m.database = value
	case stepPayment:
		m.payment = value
	case stepMail:
		m.mail = value
	case stepWorkspaces:
		m.workspaces = value == "true"
	case stepStorage:
		m.storage = value == "true"
	case stepDemo:
		m.demo = value == "true"
	}
}

func (m *model) toggleCurrent() {
	options := m.options()
	if m.cursor < 0 || m.cursor >= len(options) {
		return
	}
	value := options[m.cursor].value
	if m.step == stepOAuth {
		m.oauth[value] = !m.oauth[value]
	}
	if m.step == stepContent {
		m.content[value] = !m.content[value]
	}
}

func (m model) optionSelected(value string) bool {
	switch m.step {
	case stepEdition:
		return m.edition == value
	case stepFramework:
		return m.framework == value
	case stepDatabase:
		return m.database == value
	case stepPayment:
		return m.payment == value
	case stepMail:
		return m.mail == value
	case stepWorkspaces:
		return m.workspaces == (value == "true")
	case stepOAuth:
		return m.oauth[value]
	case stepStorage:
		return m.storage == (value == "true")
	case stepContent:
		return m.content[value]
	case stepDemo:
		return m.demo == (value == "true")
	default:
		return false
	}
}

func (m model) selectedCursor() int {
	for index, option := range m.options() {
		if m.optionSelected(option.value) {
			return index
		}
	}
	return 0
}

func (m model) progress() string {
	completed := int(m.step) + 1
	total := int(stepReview) + 1
	if m.edition == "free" {
		total = 5
		switch m.step {
		case stepEdition, stepDestination, stepModule, stepName:
			completed = int(m.step) + 1
		case stepReview:
			completed = total
		}
	}
	return fmt.Sprintf("%d/%d", completed, total)
}

func (m model) contentWidth() int {
	leftMargin := 4
	if m.width < 60 {
		leftMargin = 2
	}
	return max(32, m.width-leftMargin-2)
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func displayValue(value string) string {
	switch value {
	case "free":
		return "Free"
	case "paid":
		return "Paid"
	case "htmx":
		return "htmx"
	case "datastar":
		return "Datastar"
	case "sqlite":
		return "SQLite"
	case "postgres":
		return "PostgreSQL"
	case "stripe":
		return "Stripe"
	case "polar":
		return "Polar"
	case "smtp":
		return "SMTP"
	case "resend":
		return "Resend"
	case "google":
		return "Google"
	case "github":
		return "GitHub"
	case "blog":
		return "Blog"
	case "docs":
		return "Docs"
	default:
		return value
	}
}

func selectedValues(values map[string]bool) string {
	selected := selectedKeys(values)
	if len(selected) == 0 {
		return "None"
	}
	for index := range selected {
		selected[index] = displayValue(selected[index])
	}
	return strings.Join(selected, ", ")
}

func selectedKeys(values map[string]bool) []string {
	order := []string{"google", "github", "blog", "docs"}
	selected := make([]string, 0, len(values))
	for _, value := range order {
		if values[value] {
			selected = append(selected, value)
		}
	}
	return selected
}

func shorten(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}
