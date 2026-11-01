package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/axadrn/goilerplate/v4/internal/doctor"
	"github.com/axadrn/goilerplate/v4/internal/github"
)

type App struct {
	Out                 io.Writer
	StyledOutput        bool
	Version             string
	WorkingDirectory    string
	RunNewProjectWizard func(context.Context) ([]string, error)
	FetchReleases       func(context.Context) ([]github.Release, error)
	RunDoctor           func(context.Context, string) doctor.Report
	// FetchTemplate returns the template tree of a released version, which
	// goilerplate update compares with the current one.
	FetchTemplate func(context.Context, string) (fs.FS, error)
}

func (a *App) Run(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 {
		a.printUsage()
		return nil
	}
	switch arguments[0] {
	case "new":
		return a.newProject(ctx, arguments[1:])
	case "update":
		return a.updateProject(ctx, arguments[1:])
	case "changelog":
		return a.changelog(ctx, arguments[1:])
	case "doctor":
		return a.doctor(ctx, arguments[1:])
	case "version":
		return a.version(arguments[1:])
	case "help":
		return a.help(arguments[1:])
	case "-h", "--help":
		return a.help(nil)
	default:
		a.printUsage()
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func (a *App) help(arguments []string) error {
	if len(arguments) == 0 {
		a.printUsage()
		return nil
	}
	if len(arguments) != 1 {
		return errors.New("usage: goilerplate help [command]")
	}
	usage := map[string][]string{
		"new": {
			"goilerplate new --module <path> [options] <directory>",
			"Run without options in a terminal to open interactive setup.",
			"Renders the project on this machine. No account and no network needed.",
			"--module example.com/acme is the Go module path. Required with options.",
			"--name \"Acme\" is the project name. Defaults to the directory name.",
			"--framework htmx, datastar, svelte or headless. htmx by default.",
			"--database sqlite or postgres. sqlite by default.",
			"--payment stripe or polar. stripe by default.",
			"--mail smtp or resend. smtp by default.",
			"--workspaces adds teams with roles, invitations and seats.",
			"--oauth google,github adds sign-in providers.",
			"--storage adds S3-compatible file storage.",
			"--content blog,docs adds the Markdown blog and docs.",
			"--api adds the JSON API. --mcp adds the MCP server and needs the JSON API.",
			"--headless is the Go backend with the JSON API and no native frontend.",
			"--framework svelte is a SvelteKit app on the JSON API. It needs Node.js 22.17 or newer and pnpm.",
		},
		"update":    {"goilerplate update", "Create a Git branch containing the new generated template.", "Works for projects made with goilerplate v4 or newer."},
		"changelog": {"goilerplate changelog", "Show published GitHub release notes."},
		"doctor":    {"goilerplate doctor", "Check tools and configuration for a generated project."},
		"version":   {"goilerplate version", "Show the CLI version."},
	}
	lines, ok := usage[arguments[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", arguments[0])
	}
	if a.StyledOutput {
		a.printHeading("help")
		fmt.Fprintln(a.Out)
		for _, line := range lines {
			a.printInfo(line)
		}
		return nil
	}
	for _, line := range lines {
		fmt.Fprintln(a.Out, line)
	}
	return nil
}

func (a *App) version(arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate version")
	}
	if a.StyledOutput {
		a.printHeading("version")
		fmt.Fprintln(a.Out)
		a.printField("Version", a.Version)
		return nil
	}
	fmt.Fprintln(a.Out, a.Version)
	return nil
}

func (a *App) printUsage() {
	if a.StyledOutput {
		a.printHeading("cli")
		fmt.Fprintln(a.Out)
		a.printField("Usage", "goilerplate <command>")
		fmt.Fprintln(a.Out)
		fmt.Fprintln(a.Out, a.theme().brand.Render("Commands"))
		a.printCommand("new", "Generate a new project")
		a.printCommand("update", "Update a generated project on a new Git branch")
		a.printCommand("changelog", "Show published release notes")
		a.printCommand("doctor", "Check a generated project's local tools")
		a.printCommand("version", "Show the CLI version")
		return
	}
	fmt.Fprintln(a.Out, "goilerplate")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Usage: goilerplate <command>")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Commands:")
	fmt.Fprintln(a.Out, "  new        Generate a new project")
	fmt.Fprintln(a.Out, "  update     Update a generated project on a new Git branch")
	fmt.Fprintln(a.Out, "  changelog  Show published release notes")
	fmt.Fprintln(a.Out, "  doctor     Check a generated project's local tools")
	fmt.Fprintln(a.Out, "  version    Show the CLI version")
}
