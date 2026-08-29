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
	stepDestination step = iota
	stepModule
	stepName
	stepEdition
	stepFramework
	stepDatabase
	stepPayment
	stepMail
	stepTeams
	stepOAuth
	stepStorage
	stepContent
	stepExample
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
	panel   lipgloss.Style
}

type model struct {
	step      step
	inputs    [3]textinput.Model
	cursor    int
	width     int
	height    int
	edition   string
	framework string
	database  string
	payment   string
	mail      string
	teams     bool
	oauth     map[string]bool
	storage   bool
	content   map[string]bool
	example   bool
	submitted bool
	cancelled bool
	styles    styles
}

func newStyles(isDark bool) styles {
	color := lipgloss.LightDark(isDark)
	muted := color(lipgloss.Color("#52525B"), lipgloss.Color("#A1A1AA"))
	purple := color(lipgloss.Color("#6D28D9"), lipgloss.Color("#A78BFA"))
	cyan := color(lipgloss.Color("#0369A1"), lipgloss.Color("#67E8F9"))
	green := color(lipgloss.Color("#047857"), lipgloss.Color("#34D399"))
	border := color(lipgloss.Color("#A1A1AA"), lipgloss.Color("#52525B"))
	activeText := color(lipgloss.Color("#4C1D95"), lipgloss.Color("#FAFAFA"))
	activeBackground := color(lipgloss.Color("#EDE9FE"), lipgloss.Color("#312E81"))

	return styles{
		title:   lipgloss.NewStyle().Bold(true),
		brand:   lipgloss.NewStyle().Bold(true).Foreground(purple),
		label:   lipgloss.NewStyle().Bold(true).Foreground(cyan),
		muted:   lipgloss.NewStyle().Foreground(muted),
		value:   lipgloss.NewStyle(),
		success: lipgloss.NewStyle().Foreground(green),
		active: lipgloss.NewStyle().
			Foreground(activeText).
			Background(activeBackground).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(1, 2),
	}
}

func Run(ctx context.Context, input io.Reader, output io.Writer) ([]string, error) {
	program := tea.NewProgram(
		newModel(),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, context.Canceled) {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("run project setup: %w", err)
	}
	result, ok := final.(model)
	if !ok || result.cancelled || !result.submitted {
		return nil, ErrCancelled
	}
	return result.arguments(), nil
}

func newModel() model {
	destination := textinput.New()
	destination.Prompt = "  "
	destination.Placeholder = "./my-app"
	destination.SetValue("./my-app")
	destination.CharLimit = 240
	destination.SetWidth(48)
	destination.Focus()

	module := textinput.New()
	module.Prompt = "  "
	module.Placeholder = "example.com/my-app"
	module.SetValue("example.com/my-app")
	module.CharLimit = 240
	module.SetWidth(48)

	name := textinput.New()
	name.Prompt = "  "
	name.Placeholder = "My App"
	name.SetValue("My App")
	name.CharLimit = 100
	name.SetWidth(48)

	return model{
		inputs:    [3]textinput.Model{destination, module, name},
		width:     80,
		height:    24,
		edition:   "free",
		framework: "htmx",
		database:  "sqlite",
		payment:   "stripe",
		mail:      "smtp",
		oauth:     map[string]bool{"google": true, "github": true},
		content:   map[string]bool{},
		styles:    newStyles(true),
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
				if m.step == stepDestination {
					m.cancelled = true
					return m, tea.Quit
				}
				m.moveBack()
				return m, nil
			}
			index := int(m.step)
			updated, command := m.inputs[index].Update(message)
			m.inputs[index] = updated
			return m, command
		}

		switch key {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc", "left", "h":
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
			m.moveForward()
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.width < 40 || m.height < 20 {
		view := tea.NewView(strings.Join([]string{
			m.styles.brand.Render("goilerplate"),
			"",
			"Terminal too small.",
			"Resize to at least 40 × 20.",
			"",
			m.styles.muted.Render("ctrl+c quit"),
		}, "\n"))
		view.AltScreen = true
		view.WindowTitle = "goilerplate new"
		return view
	}
	contentWidth := min(96, max(36, m.width-4))
	header := m.styles.brand.Render("goilerplate") + "  " + m.styles.muted.Render("NEW PROJECT")
	progress := m.styles.muted.Render(m.progress())

	question := m.styles.panel.Width(min(56, contentWidth-6)).Render(m.questionView())
	body := question
	if m.width >= 92 && m.step != stepReview {
		summary := m.styles.panel.Width(28).Render(m.summaryView())
		body = lipgloss.JoinHorizontal(lipgloss.Top, question, "  ", summary)
	}

	footer := m.footerView()
	separator := "\n\n"
	if m.width < 60 {
		separator = "\n"
	}
	content := lipgloss.NewStyle().Width(contentWidth).Render(
		header + "\n" + progress + separator + body + separator + footer,
	)
	view := tea.NewView(lipgloss.NewStyle().Padding(1, 2).Render(content))
	view.AltScreen = true
	view.WindowTitle = "goilerplate new"
	return view
}

func (m model) questionView() string {
	title, hint := m.question()
	parts := []string{m.styles.title.Render(title)}
	if m.width >= 60 {
		parts = append(parts, m.styles.muted.Render(hint))
	}
	parts = append(parts, "")
	if m.isTextStep() {
		parts = append(parts, m.currentInput().View())
		return strings.Join(parts, "\n")
	}
	if m.step == stepReview {
		parts = append(parts, m.reviewView(), "", m.styles.active.Render("Generate project"))
		return strings.Join(parts, "\n")
	}
	for index, option := range m.options() {
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
		label := m.styles.value.Render(option.label)
		if index == m.cursor {
			label = m.styles.title.Render(option.label)
		}
		parts = append(parts, fmt.Sprintf("%s%s  %s", cursor, marker, label))
		if m.width >= 60 && index == m.cursor && option.description != "" {
			parts = append(parts, "     "+m.styles.muted.Render(option.description))
		}
	}
	return strings.Join(parts, "\n")
}

func (m model) reviewView() string {
	if m.width < 60 {
		rows := []string{
			m.styles.label.Render(shorten(strings.TrimSpace(m.inputs[2].Value()), 20)),
			m.styles.muted.Render("Module  " + shorten(strings.TrimSpace(m.inputs[1].Value()), 20)),
			m.styles.muted.Render("Folder  " + shorten(strings.TrimSpace(m.inputs[0].Value()), 20)),
			"",
		}
		if m.edition == "free" {
			return strings.Join(append(rows, m.styles.value.Render("Free · SQLite · SMTP · htmx")), "\n")
		}
		return strings.Join(append(rows,
			m.styles.value.Render("Paid · "+displayValue(m.framework)+" · "+displayValue(m.database)),
			m.styles.value.Render(displayValue(m.payment)+" · "+displayValue(m.mail)),
			m.styles.value.Render(fmt.Sprintf("Teams %s · OAuth %d", yesNo(m.teams), len(selectedKeys(m.oauth)))),
			m.styles.value.Render(fmt.Sprintf("Storage %s · Content %d · Example %s", yesNo(m.storage), len(selectedKeys(m.content)), yesNo(m.example))),
		), "\n")
	}
	rows := []string{
		m.styles.label.Render(shorten(strings.TrimSpace(m.inputs[2].Value()), 46)),
		m.styles.muted.Render("Module  " + shorten(strings.TrimSpace(m.inputs[1].Value()), 38)),
		m.styles.muted.Render("Folder  " + shorten(strings.TrimSpace(m.inputs[0].Value()), 38)),
		"",
	}
	if m.edition == "free" {
		return strings.Join(append(rows,
			m.styles.value.Render("Free  ·  SQLite  ·  SMTP  ·  htmx"),
			m.styles.muted.Render("No paid modules. No hidden choices."),
		), "\n")
	}
	return strings.Join(append(rows,
		m.styles.value.Render("Paid  ·  "+displayValue(m.framework)+"  ·  "+displayValue(m.database)+"  ·  "+displayValue(m.payment)+"  ·  "+displayValue(m.mail)),
		m.styles.value.Render("Teams "+yesNo(m.teams)+"  ·  OAuth "+selectedValues(m.oauth)),
		m.styles.value.Render("Storage "+yesNo(m.storage)+"  ·  Content "+selectedValues(m.content)+"  ·  Example "+yesNo(m.example)),
	), "\n")
}

func (m model) summaryView() string {
	rows := []string{
		m.styles.label.Render("Your project"),
		"",
		m.styles.value.Render(shorten(strings.TrimSpace(m.inputs[2].Value()), 20)),
		m.styles.muted.Render(shorten(strings.TrimSpace(m.inputs[1].Value()), 20)),
		m.styles.muted.Render(shorten(strings.TrimSpace(m.inputs[0].Value()), 20)),
		m.summaryRow("Edition", displayValue(m.edition)),
	}
	if m.edition == "free" {
		rows = append(rows,
			"",
			m.styles.muted.Render("SQLite · SMTP · htmx"),
			m.styles.muted.Render("No paid modules."),
		)
		return strings.Join(rows, "\n")
	}
	rows = append(rows,
		m.styles.value.Render(displayValue(m.framework)+" · "+displayValue(m.database)),
		m.styles.value.Render(displayValue(m.payment)+" · "+displayValue(m.mail)+" · Teams "+yesNo(m.teams)),
		m.styles.value.Render(fmt.Sprintf("OAuth %d · Storage %s", len(selectedKeys(m.oauth)), yesNo(m.storage))),
		m.styles.value.Render(fmt.Sprintf("Content %d · Example %s", len(selectedKeys(m.content)), yesNo(m.example))),
	)
	return strings.Join(rows, "\n")
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
		return "Choose your edition", "Free is the complete foundation. Paid unlocks every product module and updates."
	case stepFramework:
		return "Choose a frontend", "htmx is the default. Datastar uses server-sent events and fine-grained patches."
	case stepDatabase:
		return "Choose a database", "SQLite is simple. PostgreSQL is ready for distributed deployments."
	case stepPayment:
		return "Choose a payment provider", "Only the selected provider is included in your project."
	case stepMail:
		return "Choose transactional email", "SMTP works everywhere. Resend offers a hosted API."
	case stepTeams:
		return "Include team workspaces?", "Adds organizations, roles, invitations, and organization-owned billing."
	case stepOAuth:
		return "Include OAuth providers", "Choose any combination. Press Enter when ready."
	case stepStorage:
		return "Include file storage?", "Adds the S3-compatible storage package and configuration."
	case stepContent:
		return "Include content modules", "Blog and Docs share the same Markdown engine."
	case stepExample:
		return "Include the Projects example?", "A small CRUD module and matching demo data you can delete later."
	default:
		return "Ready to build", "Review the choices. The existing CLI command performs the actual generation."
	}
}

func (m model) options() []option {
	switch m.step {
	case stepEdition:
		return []option{
			{label: "Free", value: "free", description: "SQLite, SMTP, htmx, auth, security, and the app foundation."},
			{label: "Paid", value: "paid", description: "All generator choices, commercial modules, and updates."},
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
	case stepTeams, stepStorage, stepExample:
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
		if m.teams {
			arguments = append(arguments, "--teams")
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
		if m.example {
			arguments = append(arguments, "--example")
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
	return &m.inputs[int(m.step)]
}

func (m *model) moveForward() {
	if m.isTextStep() {
		m.currentInput().Blur()
	}
	if m.step == stepEdition && m.edition == "free" {
		m.setStep(stepReview)
		return
	}
	if m.step < stepReview {
		m.setStep(m.step + 1)
	}
}

func (m *model) moveBack() {
	if m.step == stepDestination {
		return
	}
	if m.step == stepReview && m.edition == "free" {
		m.setStep(stepEdition)
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
	case stepTeams:
		m.teams = value == "true"
	case stepStorage:
		m.storage = value == "true"
	case stepExample:
		m.example = value == "true"
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
	case stepTeams:
		return m.teams == (value == "true")
	case stepOAuth:
		return m.oauth[value]
	case stepStorage:
		return m.storage == (value == "true")
	case stepContent:
		return m.content[value]
	case stepExample:
		return m.example == (value == "true")
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
	if m.edition == "free" && m.step == stepReview {
		return "● ● ● ● ●   ready"
	}
	completed := int(m.step) + 1
	total := int(stepReview) + 1
	filled := strings.Repeat("● ", min(completed, total))
	empty := strings.Repeat("○ ", max(0, total-completed))
	return strings.TrimSpace(filled+empty) + fmt.Sprintf("   %d/%d", min(completed, total), total)
}

func (m model) summaryRow(label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "None"
	}
	return m.styles.muted.Render(label+":") + " " + m.styles.value.Render(value)
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
