package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/v3/internal/cli"
	"github.com/axadrn/goilerplate/v3/internal/config"
	"github.com/axadrn/goilerplate/v3/internal/desktop"
	"github.com/axadrn/goilerplate/v3/internal/doctor"
	"github.com/axadrn/goilerplate/v3/internal/github"
	"github.com/axadrn/goilerplate/v3/internal/service"
	"github.com/axadrn/goilerplate/v3/internal/tui"
	"github.com/charmbracelet/x/term"
)

const githubClientID = "Ov23liBIZFEy7FHommLT"

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
	store, err := config.DefaultStore()
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	application := &cli.App{
		Out:            os.Stdout,
		StyledOutput:   styledTerminal(os.Stdout),
		Store:          store,
		Device:         github.NewDeviceClient(httpClient),
		GitHubClientID: githubClientID,
		DefaultAPIURL:  "https://goilerplate.com",
		APIURLOverride: strings.TrimSpace(os.Getenv("GOILERPLATE_API_URL")),
		Version:        buildVersion(),
		FetchReleases: func(ctx context.Context) ([]github.Release, error) {
			return github.ListReleases(ctx, httpClient)
		},
		RunDoctor: doctor.Inspect,
		NewService: func(baseURL string) (cli.ServiceClient, error) {
			return service.NewClient(baseURL, httpClient)
		},
	}
	if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		application.OpenBrowser = desktop.OpenBrowser
		application.CopyToClipboard = desktop.CopyToClipboard
		application.RunNewProjectWizard = func(ctx context.Context, paidAccess bool) (cli.ProjectWizardResult, error) {
			result, err := tui.Run(ctx, os.Stdin, os.Stdout, paidAccess)
			return cli.ProjectWizardResult{Arguments: result.Arguments, OpenPricing: result.OpenPricing}, err
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
