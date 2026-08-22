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
	"github.com/axadrn/goilerplate/v3/internal/project"
)

func (a *App) newProject(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 && a.RunNewProjectWizard != nil {
		var err error
		arguments, err = a.RunNewProjectWizard(ctx)
		if err != nil {
			return err
		}
	}

	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(a.Out)
	name := flags.String("name", "", "project name")
	modulePath := flags.String("module", "", "Go module path")
	edition := flags.String("edition", "free", "free or paid")
	framework := flags.String("framework", "htmx", "htmx or datastar")
	database := flags.String("database", "sqlite", "sqlite or postgres")
	payment := flags.String("payment", "", "stripe, polar, or lemon_squeezy")
	mail := flags.String("mail", "smtp", "smtp or resend")
	teams := flags.Bool("teams", false, "include team workspaces")
	oauth := flags.String("oauth", "", "comma-separated OAuth providers")
	storage := flags.Bool("storage", false, "include file storage")
	content := flags.String("content", "", "comma-separated blog and docs modules")
	example := flags.Bool("example", false, "include the Projects example")
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
		Teams:       *teams,
		OAuth:       splitList(*oauth),
		Storage:     *storage,
		Content:     splitList(*content),
		Example:     *example,
	}
	if err := validateEditionSelection(answers); err != nil {
		return err
	}

	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	authorizationToken := strings.TrimSpace(a.MachineToken)
	if authorizationToken == "" {
		if configuration.SessionToken == "" {
			if err := a.login(ctx, nil); err != nil {
				return err
			}
			configuration, err = a.Store.Load()
			if err != nil {
				return err
			}
		}
		authorizationToken = configuration.SessionToken
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return err
	}
	if strings.TrimSpace(a.MachineToken) == "" {
		activation, err := client.ActivationStatus(ctx, authorizationToken)
		if err != nil {
			return err
		}
		switch activation.State {
		case api.ActivationStateActive:
		case api.ActivationStatePending:
			fmt.Fprintln(a.Out, "Check your email to activate Free access. Waiting for confirmation...")
			if err := a.waitForActivation(ctx, client, authorizationToken); err != nil {
				return err
			}
			fmt.Fprintln(a.Out, "Free access activated")
		case api.ActivationStateResendRequired:
			return errActivationResendRequired
		case api.ActivationStateUnavailable:
			return errActivationUnavailable
		default:
			return fmt.Errorf("unknown activation state %q", activation.State)
		}
	}
	archive, err := os.CreateTemp("", "goilerplate-project-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create project download: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	defer archive.Close()

	templateVersion := ""
	if a.Version != "" && a.Version != "dev" {
		templateVersion = a.Version
	}
	generatedVersion, err := client.Generate(ctx, authorizationToken, api.GenerateRequest{
		TemplateVersion: templateVersion,
		Answers:         answers,
	}, archive)
	if err != nil {
		return err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("read project download: %w", err)
	}
	if err := project.Extract(archive, destination); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Created %s in %s with %s\n", answers.ProjectName, destination, generatedVersion)
	return nil
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
		if answers.Framework != "htmx" || answers.Database != "sqlite" || answers.Payment != "none" || answers.Mail != "smtp" || answers.Teams || len(answers.OAuth) != 0 || answers.Storage || len(answers.Content) != 0 || answers.Example {
			return errors.New("Free uses SQLite, SMTP, htmx, and no paid modules")
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
