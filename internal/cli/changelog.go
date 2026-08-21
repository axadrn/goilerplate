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
		fmt.Fprintln(a.Out, "No releases published yet")
		return nil
	}
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
		fmt.Fprintln(a.Out, name)
		if !release.PublishedAt.IsZero() {
			fmt.Fprintln(a.Out, release.PublishedAt.Format("2006-01-02"))
		}
		if body := strings.TrimSpace(safeTerminalText(release.Body)); body != "" {
			fmt.Fprintln(a.Out, body)
		}
	}
	fmt.Fprintf(a.Out, "\nShowing up to %d latest releases. View all: %s\n", github.ReleaseLimit, github.ReleasesPageURL)
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
