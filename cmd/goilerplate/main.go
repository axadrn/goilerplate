package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/v4/internal/cli"
	"github.com/axadrn/goilerplate/v4/internal/doctor"
	"github.com/axadrn/goilerplate/v4/internal/github"
	"github.com/axadrn/goilerplate/v4/internal/tui"
	"github.com/axadrn/goilerplate/v4/internal/update"
	"github.com/charmbracelet/x/term"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		if errors.Is(err, tui.ErrCancelled) {
			return
		}
		fmt.Fprintln(os.Stderr, cli.RenderError(err, styledTerminal(os.Stderr)))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	application := &cli.App{
		Out:          os.Stdout,
		StyledOutput: styledTerminal(os.Stdout),
		Version:      buildVersion(),
		FetchReleases: func(ctx context.Context) ([]github.Release, error) {
			return github.ListReleases(ctx, httpClient)
		},
		RunDoctor: doctor.Inspect,
		FetchTemplate: func(ctx context.Context, version string) (fs.FS, error) {
			return update.FetchTemplate(ctx, httpClient, moduleProxy(os.Getenv("GOPROXY")), version)
		},
	}
	if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		application.RunNewProjectWizard = func(ctx context.Context) ([]string, error) {
			return tui.Run(ctx, os.Stdin, os.Stdout)
		}
	}
	return application.Run(ctx, os.Args[1:])
}

func isTerminal(file *os.File) bool {
	return term.IsTerminal(file.Fd())
}

func styledTerminal(file *os.File) bool {
	_, noColor := os.LookupEnv("NO_COLOR")
	return isTerminal(file) && !noColor
}

// moduleProxy returns the first HTTP proxy of GOPROXY, as the go command would
// try it, or the public Go module proxy.
func moduleProxy(goproxy string) string {
	for _, entry := range strings.FieldsFunc(goproxy, func(r rune) bool { return r == ',' || r == '|' }) {
		entry = strings.TrimSpace(entry)
		if strings.HasPrefix(entry, "https://") || strings.HasPrefix(entry, "http://") {
			return entry
		}
		break
	}
	return update.DefaultProxy
}

func buildVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
