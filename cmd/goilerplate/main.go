package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/internal/cli"
	"github.com/axadrn/goilerplate/internal/config"
	"github.com/axadrn/goilerplate/internal/github"
	"github.com/axadrn/goilerplate/internal/service"
	"github.com/axadrn/goilerplate/internal/tui"
	"github.com/charmbracelet/x/term"
)

var (
	version        = "dev"
	githubClientID = ""
)

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
	clientID := strings.TrimSpace(os.Getenv("GOILERPLATE_GITHUB_CLIENT_ID"))
	if clientID == "" {
		clientID = githubClientID
	}
	apiURL := strings.TrimSpace(os.Getenv("GOILERPLATE_API_URL"))
	if apiURL == "" {
		apiURL = "https://goilerplate.com"
	}
	application := &cli.App{
		Out:            os.Stdout,
		Store:          store,
		Device:         github.NewDeviceClient(httpClient),
		GitHubClientID: clientID,
		DefaultAPIURL:  apiURL,
		MachineToken:   strings.TrimSpace(os.Getenv("GOILERPLATE_TOKEN")),
		Version:        version,
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
