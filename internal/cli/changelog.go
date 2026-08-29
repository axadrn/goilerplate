package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/axadrn/goilerplate/v3/internal/github"
)

func (a *App) changelog(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate changelog")
	}
	if a.FetchReleases == nil {
		return errors.New("release notes are not configured")
	}
	releases, err := a.FetchReleases(ctx)
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		a.printInfo("No releases published yet")
		return nil
	}
	if a.StyledOutput {
		a.printHeading("changelog")
		fmt.Fprintln(a.Out)
	}
	theme := a.theme()
	for index, release := range releases {
		if index > 0 {
			fmt.Fprintln(a.Out)
		}
		name := strings.TrimSpace(release.Name)
		if name == "" || name == release.Tag {
			name = release.Tag
		} else {
			name = release.Tag + ": " + name
		}
		if release.Prerelease {
			name += " (prerelease)"
		}
		if a.StyledOutput {
			fmt.Fprintln(a.Out, theme.accent.Render(name))
		} else {
			fmt.Fprintln(a.Out, name)
		}
		if !release.PublishedAt.IsZero() {
			date := release.PublishedAt.Format("2006-01-02")
			if a.StyledOutput {
				fmt.Fprintln(a.Out, theme.muted.Render(date))
			} else {
				fmt.Fprintln(a.Out, date)
			}
		}
		if body := strings.TrimSpace(safeTerminalText(release.Body)); body != "" {
			if a.StyledOutput {
				fmt.Fprintln(a.Out, theme.value.Render(body))
			} else {
				fmt.Fprintln(a.Out, body)
			}
		}
	}
	fmt.Fprintln(a.Out)
	a.printInfo(fmt.Sprintf("Showing up to %d latest releases. View all: %s", github.ReleaseLimit, github.ReleasesPageURL))
	return nil
}

func safeTerminalText(value string) string {
	return strings.Map(func(character rune) rune {
		if character == '\n' || character == '\t' {
			return character
		}
		if character < 0x20 || character >= 0x7f && character <= 0x9f {
			return -1
		}
		return character
	}, value)
}
