package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type outputTheme struct {
	brand   lipgloss.Style
	accent  lipgloss.Style
	success lipgloss.Style
	warning lipgloss.Style
	danger  lipgloss.Style
	muted   lipgloss.Style
	label   lipgloss.Style
	value   lipgloss.Style
}

func newOutputTheme(styled bool) outputTheme {
	if !styled {
		return outputTheme{}
	}
	return outputTheme{
		brand:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")),
		accent:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0ABFC")),
		success: lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")),
		warning: lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")),
		danger:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FB7185")),
		muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A1A1AA")),
		label:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A1A1AA")),
		value:   lipgloss.NewStyle(),
	}
}

func (a *App) theme() outputTheme {
	return newOutputTheme(a.StyledOutput)
}

func (a *App) printHeading(title string) {
	if !a.StyledOutput {
		fmt.Fprintln(a.Out, title)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, theme.brand.Render("goilerplate")+theme.muted.Render(" "+title))
}

func (a *App) printSuccess(message string) {
	if !a.StyledOutput {
		fmt.Fprintln(a.Out, message)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, theme.success.Render("✓")+" "+theme.value.Render(message))
}

func (a *App) printInfo(message string) {
	if !a.StyledOutput {
		fmt.Fprintln(a.Out, message)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, theme.accent.Render("›")+" "+theme.muted.Render(message))
}

func (a *App) printWarning(message string) {
	if !a.StyledOutput {
		fmt.Fprintln(a.Out, message)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, theme.warning.Render("!")+" "+theme.value.Render(message))
}

func (a *App) printField(label, value string) {
	if !a.StyledOutput {
		fmt.Fprintf(a.Out, "%s: %s\n", label, value)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, theme.label.Render(fmt.Sprintf("%-10s", label))+theme.value.Render(value))
}

func (a *App) printCommand(command, description string) {
	if !a.StyledOutput {
		fmt.Fprintf(a.Out, "  %-15s %s\n", command, description)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, "  "+theme.accent.Render(fmt.Sprintf("%-15s", command))+theme.muted.Render(description))
}

func (a *App) printListItem(value string) {
	if !a.StyledOutput {
		fmt.Fprintf(a.Out, "  %s\n", value)
		return
	}
	theme := a.theme()
	fmt.Fprintln(a.Out, "  "+theme.accent.Render("•")+" "+theme.muted.Render(value))
}

func RenderError(err error, styled bool) string {
	if !styled {
		return "Error: " + err.Error()
	}
	theme := newOutputTheme(true)
	return theme.danger.Render("✕") + " " + theme.value.Render(err.Error())
}

func renderColumns(theme outputTheme, styled bool, values ...string) string {
	if !styled {
		return "  " + strings.Join(values, "  ")
	}
	styledValues := make([]string, 0, len(values))
	for index, value := range values {
		style := theme.value
		if index > 0 {
			style = theme.muted
		}
		styledValues = append(styledValues, style.Render(value))
	}
	return "  " + strings.Join(styledValues, theme.muted.Render("  "))
}
