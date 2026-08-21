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
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	store, err := config.DefaultStore()
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	apiURL := strings.TrimSpace(os.Getenv("GOILERPLATE_API_URL"))
	if apiURL == "" {
		apiURL = "https://goilerplate.com"
	}
	application := &cli.App{
		Out:            os.Stdout,
		Store:          store,
		Device:         github.NewDeviceClient(httpClient),
		GitHubClientID: githubClientID,
		DefaultAPIURL:  apiURL,
		MachineToken:   strings.TrimSpace(os.Getenv("GOILERPLATE_TOKEN")),
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
		application.RunNewProjectWizard = func(ctx context.Context) ([]string, error) {
			return tui.Run(ctx, os.Stdin, os.Stdout)
		}
	}
	return application.Run(ctx, os.Args[1:])
}

func isTerminal(file *os.File) bool {
	return term.IsTerminal(file.Fd())
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
