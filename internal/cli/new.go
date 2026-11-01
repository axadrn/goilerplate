package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/axadrn/goilerplate/v4/generator"
	"github.com/axadrn/goilerplate/v4/internal/project"
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
	if errors.Is(err, flag.ErrHelp) {
		return a.help([]string{"new"})
	}
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
	answers := generator.Config{
		ProjectName: strings.TrimSpace(*name),
		ModulePath:  strings.TrimSpace(*modulePath),
		Framework:   generator.Framework(strings.TrimSpace(*framework)),
		Database:    generator.Database(strings.TrimSpace(*database)),
		Payment:     generator.Payment(strings.TrimSpace(*payment)),
		Mail:        generator.Mail(strings.TrimSpace(*mail)),
		Workspaces:  *workspaces,
		OAuth:       splitList[generator.OAuth](*oauth),
		Storage:     *storage,
		Content:     splitList[generator.Content](*content),
		API:         *jsonAPI,
		MCP:         *mcp,
	}
	err = validateSelection(answers)
	if err != nil {
		return err
	}

	version := a.templateVersion()
	template, err := generator.CurrentTemplate(version)
	if err != nil {
		return err
	}
	var archive bytes.Buffer
	if err := template.WriteArchive(&archive, answers); err != nil {
		return err
	}
	if err := project.Create(ctx, &archive, destination); err != nil {
		return err
	}
	a.printSuccess(fmt.Sprintf("Created %s in %s with %s", answers.ProjectName, destination, version))
	a.printInfo("Git is ready on main with an initial commit")
	return nil
}

// devTemplateVersion names the template of a build without a release
// version, such as go run or go build from a checkout.
const devTemplateVersion = "v4.0.0-dev"

// templateVersion is the version written to goilerplate.lock. The template is
// embedded in the CLI, so it is the version of the CLI itself.
func (a *App) templateVersion() string {
	if generator.ValidateVersion(a.Version) == nil {
		return a.Version
	}
	return devTemplateVersion
}

func splitList[T ~string](value string) []T {
	if strings.TrimSpace(value) == "" {
		return []T{}
	}
	parts := strings.Split(value, ",")
	result := make([]T, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, T(part))
		}
	}
	return result
}

func validateSelection(answers generator.Config) error {
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
