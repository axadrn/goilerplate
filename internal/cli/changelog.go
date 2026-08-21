package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		fmt.Fprintln(a.Out, name)
		if !release.PublishedAt.IsZero() {
			fmt.Fprintln(a.Out, release.PublishedAt.Format("2006-01-02"))
		}
		if body := strings.TrimSpace(release.Body); body != "" {
			fmt.Fprintln(a.Out, body)
		}
	}
	return nil
}
