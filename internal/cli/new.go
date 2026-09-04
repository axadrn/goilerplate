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
		configuration, client, err = a.projectClient(ctx)
		if err != nil {
			return err
		}
		identity, err := client.WhoAmI(ctx, configuration.SessionToken)
		if err != nil {
			return err
		}
		result, err := a.RunNewProjectWizard(ctx, hasPaidAccess(identity))
		if err != nil {
			return err
		}
		if result.OpenPricing {
			a.openPricing(ctx, configuration)
			return nil
		}
		arguments = result.Arguments
	}

	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "project name")
	modulePath := flags.String("module", "", "Go module path")
	edition := flags.String("edition", "free", "free or paid")
	framework := flags.String("framework", "htmx", "htmx or datastar")
	database := flags.String("database", "sqlite", "sqlite or postgres")
	payment := flags.String("payment", "", "stripe or polar")
	mail := flags.String("mail", "smtp", "smtp or resend")
	workspaces := flags.Bool("workspaces", false, "include shared workspaces")
	oauth := flags.String("oauth", "", "comma-separated OAuth providers")
	storage := flags.Bool("storage", false, "include file storage")
	content := flags.String("content", "", "comma-separated blog and docs modules")
	if err := flags.Parse(arguments); err != nil {
		return err
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
	if *payment == "" {
		if *edition == "free" {
			*payment = "none"
		} else {
			*payment = "stripe"
		}
	}
	answers := api.GenerationAnswers{
		ProjectName: strings.TrimSpace(*name),
		ModulePath:  strings.TrimSpace(*modulePath),
		Edition:     strings.TrimSpace(*edition),
		Framework:   strings.TrimSpace(*framework),
		Database:    strings.TrimSpace(*database),
		Payment:     strings.TrimSpace(*payment),
		Mail:        strings.TrimSpace(*mail),
		Workspaces:  *workspaces,
		OAuth:       splitList(*oauth),
		Storage:     *storage,
		Content:     splitList(*content),
	}
	if err := validateEditionSelection(answers); err != nil {
		return err
	}

	if client == nil {
		var err error
		configuration, client, err = a.projectClient(ctx)
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

func (a *App) openPricing(ctx context.Context, configuration config.Config) {
	pricingURL := strings.TrimRight(a.apiURL(configuration), "/") + "/pricing"
	opened := a.OpenBrowser != nil && a.OpenBrowser(ctx, pricingURL) == nil
	if a.StyledOutput {
		a.printHeading("paid")
		fmt.Fprintln(a.Out)
	}
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

func hasPaidAccess(identity api.WhoAmIResponse) bool {
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

func validateEditionSelection(answers api.GenerationAnswers) error {
	if answers.Framework != "htmx" && answers.Framework != "datastar" {
		return fmt.Errorf("unsupported framework %q", answers.Framework)
	}
	switch answers.Edition {
	case "free":
		if (answers.Framework != "htmx" && answers.Framework != "datastar") || answers.Database != "sqlite" || answers.Payment != "none" || answers.Mail != "smtp" || answers.Workspaces || len(answers.OAuth) != 0 || answers.Storage || len(answers.Content) != 0 {
			return errors.New("Free uses SQLite, SMTP, htmx or datastar, and no paid modules")
		}
	case "paid":
		if answers.Payment == "none" {
			return errors.New("Paid requires a payment provider")
		}
	default:
		return fmt.Errorf("unsupported edition %q", answers.Edition)
	}
	return nil
}
