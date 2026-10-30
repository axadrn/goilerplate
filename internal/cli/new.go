package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/config"
	"github.com/axadrn/goilerplate/v3/internal/project"
)

func (a *App) newProject(ctx context.Context, arguments []string) error {
	interactive := len(arguments) == 0 && a.RunNewProjectWizard != nil
	var configuration config.Config
	var client ServiceClient
	if interactive {
		var err error
		configuration, client, err = a.licensedClient(ctx)
		if err != nil {
			return err
		}
		arguments, err = a.RunNewProjectWizard(ctx)
		if err != nil {
			return err
		}
	}

	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "project name")
	modulePath := flags.String("module", "", "Go module path")
	framework := flags.String("framework", "htmx", "htmx, datastar, svelte or headless")
	database := flags.String("database", "sqlite", "sqlite or postgres")
	payment := flags.String("payment", "stripe", "stripe or polar")
	mail := flags.String("mail", "smtp", "smtp or resend")
	workspaces := flags.Bool("workspaces", false, "include shared workspaces")
	oauth := flags.String("oauth", "", "comma-separated OAuth providers")
	storage := flags.Bool("storage", false, "include file storage")
	content := flags.String("content", "", "comma-separated blog and docs modules")
	jsonAPI := flags.Bool("api", false, "include the JSON API")
	headless := flags.Bool("headless", false, "Go backend with the JSON API and no native frontend")
	mcp := flags.Bool("mcp", false, "include the MCP server, needs the JSON API")
	err := flags.Parse(arguments)
	if err != nil {
		return err
	}
	if *headless {
		frameworkSet := false
		flags.Visit(func(set *flag.Flag) {
			if set.Name == "framework" {
				frameworkSet = true
			}
		})
		if frameworkSet {
			return errors.New("--headless cannot be combined with --framework")
		}
		*framework = "headless"
	}
	switch strings.TrimSpace(*framework) {
	case "headless", "svelte":
		*jsonAPI = true
	}
	if flags.NArg() != 1 {
		return errors.New("usage: goilerplate new [options] <directory>")
	}
	destination := flags.Arg(0)
	if strings.TrimSpace(*modulePath) == "" {
		return errors.New("--module is required")
	}
	if strings.TrimSpace(*name) == "" {
		*name = filepath.Base(filepath.Clean(destination))
	}
	answers := api.GenerationAnswers{
		ProjectName: strings.TrimSpace(*name),
		ModulePath:  strings.TrimSpace(*modulePath),
		Edition:     api.EditionPaid,
		Framework:   strings.TrimSpace(*framework),
		Database:    strings.TrimSpace(*database),
		Payment:     strings.TrimSpace(*payment),
		Mail:        strings.TrimSpace(*mail),
		Workspaces:  *workspaces,
		OAuth:       splitList(*oauth),
		Storage:     *storage,
		Content:     splitList(*content),
		API:         *jsonAPI,
		MCP:         *mcp,
	}
	err = validateSelection(answers)
	if err != nil {
		return err
	}

	if client == nil {
		configuration, client, err = a.licensedClient(ctx)
		if err != nil {
			return err
		}
	}
	archive, err := os.CreateTemp("", "goilerplate-project-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create project download: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	defer archive.Close()

	generatedVersion, err := client.Generate(ctx, configuration.SessionToken, api.GenerateRequest{
		Answers: answers,
	}, archive)
	if err != nil {
		return err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("read project download: %w", err)
	}
	if err := project.Create(ctx, archive, destination); err != nil {
		return err
	}
	a.printSuccess(fmt.Sprintf("Created %s in %s with %s", answers.ProjectName, destination, generatedVersion))
	a.printInfo("Git is ready on main with an initial commit")
	return nil
}

func (a *App) projectClient(ctx context.Context) (config.Config, ServiceClient, error) {
	configuration, err := a.Store.Load()
	if err != nil {
		return config.Config{}, nil, err
	}
	if configuration.SessionToken == "" {
		if err := a.login(ctx, nil); err != nil {
			return config.Config{}, nil, err
		}
		configuration, err = a.Store.Load()
		if err != nil {
			return config.Config{}, nil, err
		}
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return config.Config{}, nil, err
	}
	return configuration, client, nil
}

// licensedClient returns a client for an account with an active license.
// Without one it opens pricing and fails, so nothing is asked or generated.
func (a *App) licensedClient(ctx context.Context) (config.Config, ServiceClient, error) {
	configuration, client, err := a.projectClient(ctx)
	if err != nil {
		return config.Config{}, nil, err
	}
	identity, err := client.WhoAmI(ctx, configuration.SessionToken)
	if err != nil {
		return config.Config{}, nil, err
	}
	if !hasActiveLicense(identity) {
		a.openPricing(ctx, configuration)
		return config.Config{}, nil, errors.New("no active license")
	}
	return configuration, client, nil
}

func (a *App) pricingURL(configuration config.Config) string {
	return strings.TrimRight(a.apiURL(configuration), "/") + "/pricing"
}

func (a *App) openPricing(ctx context.Context, configuration config.Config) {
	pricingURL := a.pricingURL(configuration)
	opened := a.OpenBrowser != nil && a.OpenBrowser(ctx, pricingURL) == nil
	if a.StyledOutput {
		a.printHeading("license")
		fmt.Fprintln(a.Out)
	}
	a.printInfo("goilerplate needs a license")
	if opened {
		a.printSuccess("Pricing opened in your browser")
	} else {
		a.printField("Open pricing", pricingURL)
	}
	if a.StyledOutput {
		a.printInfo("Buy with your verified GitHub email")
		a.printInfo("Then run goilerplate new again. No new login needed")
		return
	}
	fmt.Fprintln(a.Out, "Buy with your verified GitHub email, then run goilerplate new again. No new login needed")
}

func hasActiveLicense(identity api.WhoAmIResponse) bool {
	for _, license := range identity.Licenses {
		if license.Status == api.LicenseStatusActive {
			return true
		}
	}
	return false
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func validateSelection(answers api.GenerationAnswers) error {
	switch answers.Framework {
	case "htmx", "datastar":
	case "headless":
		if !answers.API {
			return errors.New("Headless requires the JSON API")
		}
		if len(answers.Content) != 0 {
			return errors.New("Headless does not support blog or docs")
		}
	case "svelte":
		if !answers.API {
			return errors.New("SvelteKit requires the JSON API")
		}
		if len(answers.Content) != 0 {
			return errors.New("SvelteKit does not support blog or docs")
		}
	default:
		return fmt.Errorf("unsupported framework %q", answers.Framework)
	}
	if answers.MCP && !answers.API {
		return errors.New("MCP requires the JSON API")
	}
	switch answers.Payment {
	case "stripe", "polar":
	default:
		return fmt.Errorf("unsupported payment provider %q, use stripe or polar", answers.Payment)
	}
	return nil
}
