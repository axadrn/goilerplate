package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/axadrn/goilerplate/v4/generator"
	"github.com/axadrn/goilerplate/v4/internal/doctor"
	"github.com/axadrn/goilerplate/v4/internal/github"
	"github.com/axadrn/goilerplate/v4/internal/project"
	projectupdate "github.com/axadrn/goilerplate/v4/internal/update"
)

func TestHelpCoversEveryCommand(t *testing.T) {
	for _, command := range []string{"new", "update", "changelog", "doctor", "version"} {
		t.Run(command, func(t *testing.T) {
			output := &bytes.Buffer{}
			app := testApp(output)
			if err := app.Run(context.Background(), []string{"help", command}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "goilerplate "+command) {
				t.Fatalf("output = %q", output.String())
			}
		})
	}
}

func TestUnknownCommandPrintsUsage(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	if err := app.Run(context.Background(), []string{"wat"}); err == nil {
		t.Fatal("unknown command succeeded")
	}
	if !strings.Contains(output.String(), "Usage: goilerplate <command>") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestChangelogPrintsPublishedReleaseNotes(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	app.FetchReleases = func(context.Context) ([]github.Release, error) {
		return []github.Release{{
			Tag: "v3.0.0-beta.1", Name: "Beta", Body: "Everything\x1b[31m you need.\x07",
			PublishedAt: time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), Prerelease: true,
		}}, nil
	}
	if err := app.Run(context.Background(), []string{"changelog"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "v3.0.0-beta.1: Beta (prerelease)") || !strings.Contains(got, "2026-08-21") || !strings.Contains(got, "Everything[31m you need.") || !strings.Contains(got, "Showing up to 10 latest releases") || strings.ContainsRune(got, '\x1b') || strings.ContainsRune(got, '\x07') {
		t.Fatalf("output = %q", got)
	}
}

func TestDoctorPrintsChecksAndFailsOnErrors(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	app.WorkingDirectory = "/project"
	app.RunDoctor = func(_ context.Context, directory string) doctor.Report {
		if directory != "/project" {
			t.Fatalf("directory = %q", directory)
		}
		return doctor.Report{
			Checks: []doctor.Check{
				{Name: "go", Message: "1.25.7", Level: doctor.LevelOK},
				{Name: ".env", Message: "missing", Level: doctor.LevelWarning},
				{Name: "git", Message: "too old", Level: doctor.LevelError},
			},
			Errors: 1,
		}
	}
	if err := app.Run(context.Background(), []string{"doctor"}); err == nil {
		t.Fatal("doctor accepted failed report")
	}
	if got := output.String(); !strings.Contains(got, "[OK] go") || !strings.Contains(got, "[WARN] .env") || !strings.Contains(got, "[FAIL] git") {
		t.Fatalf("output = %q", got)
	}
}

func TestNewRendersTheProjectLocally(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	app.Version = "v4.0.0-beta.1"
	destination := filepath.Join(t.TempDir(), "acme")

	err := app.Run(context.Background(), []string{
		"new",
		"--name", "Acme",
		"--module", "example.com/acme",
		"--framework", "datastar",
		"--database", "postgres",
		"--workspaces",
		"--oauth", "google,github",
		destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "go.mod"))
	if err != nil || !strings.HasPrefix(string(content), "module example.com/acme\n") {
		t.Fatalf("go.mod = %q, %v", content, err)
	}
	version, answers := readProjectLock(t, destination)
	if version != "v4.0.0-beta.1" {
		t.Fatalf("template version = %q", version)
	}
	if answers.ProjectName != "Acme" || answers.Framework != "datastar" || answers.Database != "postgres" || answers.Payment != "stripe" || !answers.Workspaces || !slices.Equal(answers.OAuth, []generator.OAuth{generator.OAuthGoogle, generator.OAuthGitHub}) {
		t.Fatalf("answers = %#v", answers)
	}
	if !strings.Contains(output.String(), "Created Acme") || !strings.Contains(output.String(), "with v4.0.0-beta.1") {
		t.Fatalf("output = %q", output.String())
	}
	if branch := strings.TrimSpace(runCLIGit(t, destination, "branch", "--show-current")); branch != "main" {
		t.Fatalf("branch = %q", branch)
	}
	if message := strings.TrimSpace(runCLIGit(t, destination, "log", "-1", "--format=%s")); message != "Initial commit" {
		t.Fatalf("commit message = %q", message)
	}
	if status := strings.TrimSpace(runCLIGit(t, destination, "status", "--porcelain")); status != "" {
		t.Fatalf("worktree = %q", status)
	}
	if !strings.Contains(output.String(), "Git is ready on main") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestNewMatchesTheGenerator(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	app.Version = "v4.0.0"
	destination := filepath.Join(t.TempDir(), "acme")
	err := app.Run(context.Background(), []string{
		"new", "--name", "Acme", "--module", "example.com/acme",
		"--payment", "polar", "--mail", "resend", "--storage", "--content", "blog,docs", "--api", "--mcp",
		destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(t.TempDir(), "want")
	err = generator.Generate(want, "v4.0.0", generator.Config{
		ProjectName: "Acme",
		ModulePath:  "example.com/acme",
		Framework:   generator.FrameworkHTMX,
		Database:    generator.DatabaseSQLite,
		Payment:     generator.PaymentPolar,
		Mail:        generator.MailResend,
		Storage:     true,
		Content:     []generator.Content{generator.ContentBlog, generator.ContentDocs},
		API:         true,
		MCP:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, wanted := readCLITree(t, destination), readCLITree(t, want)
	if len(got) != len(wanted) {
		t.Fatalf("new wrote %d files, the generator %d", len(got), len(wanted))
	}
	for name, content := range wanted {
		if got[name] != content {
			t.Fatalf("%s differs from the generator", name)
		}
	}
}

func TestNewWithoutAReleaseVersionUsesTheDevVersion(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	destination := filepath.Join(t.TempDir(), "acme")
	if err := app.Run(context.Background(), []string{"new", "--module", "example.com/acme", destination}); err != nil {
		t.Fatal(err)
	}
	if version, _ := readProjectLock(t, destination); version != devTemplateVersion {
		t.Fatalf("template version = %q", version)
	}
}
func TestNewWithoutArgumentsUsesTheWizard(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	destination := filepath.Join(t.TempDir(), "acme")
	wizardCalled := false
	app.RunNewProjectWizard = func(context.Context) ([]string, error) {
		wizardCalled = true
		return []string{
			"--name", "Acme",
			"--module", "example.com/acme",
			"--database", "postgres",
			"--workspaces",
			destination,
		}, nil
	}

	if err := app.Run(context.Background(), []string{"new"}); err != nil {
		t.Fatal(err)
	}
	if !wizardCalled {
		t.Fatal("wizard was not called")
	}
	if _, answers := readProjectLock(t, destination); answers.Database != "postgres" || !answers.Workspaces {
		t.Fatalf("answers = %#v", answers)
	}
}
func TestNewWizardHeadlessArgumentsReachTheGenerator(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	destination := filepath.Join(t.TempDir(), "acme")
	app.RunNewProjectWizard = func(context.Context) ([]string, error) {
		return []string{
			"--name", "Acme",
			"--module", "example.com/acme",
			"--headless",
			destination,
		}, nil
	}

	err := app.Run(context.Background(), []string{"new"})
	if err != nil {
		t.Fatal(err)
	}
	_, answers := readProjectLock(t, destination)
	if answers.Framework != "headless" || !answers.API || answers.Payment != "stripe" {
		t.Fatalf("answers = %#v", answers)
	}
}
func TestNewReturnsWizardError(t *testing.T) {
	wizardError := errors.New("cancelled")
	app := testApp(&bytes.Buffer{})
	app.RunNewProjectWizard = func(context.Context) ([]string, error) {
		return nil, wizardError
	}

	if err := app.Run(context.Background(), []string{"new"}); !errors.Is(err, wizardError) {
		t.Fatalf("new error = %v", err)
	}
}

func TestNewWithoutArgumentsRequiresInteractiveTerminal(t *testing.T) {
	app := testApp(&bytes.Buffer{})

	err := app.Run(context.Background(), []string{"new"})
	if err == nil || !strings.Contains(err.Error(), "usage: goilerplate new") {
		t.Fatalf("new error = %v", err)
	}
}

func TestNewRejectsUnknownPaymentBeforeGenerating(t *testing.T) {
	for _, payment := range []string{"none", "paddle"} {
		app := testApp(&bytes.Buffer{})
		err := app.Run(context.Background(), []string{
			"new", "--module", "example.com/acme", "--payment", payment, filepath.Join(t.TempDir(), "acme"),
		})
		if err == nil || !strings.Contains(err.Error(), "use stripe or polar") {
			t.Fatalf("new --payment %s error = %v", payment, err)
		}
	}
}

func TestNewRejectsTheRemovedEditionFlag(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	err := app.Run(context.Background(), []string{
		"new", "--module", "example.com/acme", "--edition", "free", filepath.Join(t.TempDir(), "acme"),
	})
	if err == nil || !strings.Contains(err.Error(), "-edition") {
		t.Fatalf("new error = %v", err)
	}
}

func TestNewSendsHeadlessAndTheJSONAPI(t *testing.T) {
	for _, test := range []struct {
		name          string
		arguments     []string
		wantFramework string
		wantPayment   string
		wantAPI       bool
	}{
		{name: "headless", arguments: []string{"--payment", "stripe", "--headless"}, wantFramework: "headless", wantPayment: "stripe", wantAPI: true},
		{name: "framework headless implies the JSON API", arguments: []string{"--framework", "headless"}, wantFramework: "headless", wantPayment: "stripe", wantAPI: true},
		{name: "svelte implies the JSON API", arguments: []string{"--framework", "svelte"}, wantFramework: "svelte", wantPayment: "stripe", wantAPI: true},
		{name: "svelte with polar", arguments: []string{"--payment", "polar", "--framework", "svelte", "--workspaces"}, wantFramework: "svelte", wantPayment: "polar", wantAPI: true},
		{name: "htmx with the JSON API", arguments: []string{"--api"}, wantFramework: "htmx", wantPayment: "stripe", wantAPI: true},
		{name: "datastar with the JSON API", arguments: []string{"--framework", "datastar", "--api"}, wantFramework: "datastar", wantPayment: "stripe", wantAPI: true},
		{name: "htmx without the JSON API", arguments: []string{}, wantFramework: "htmx", wantPayment: "stripe", wantAPI: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			answers := runNew(t, test.arguments)
			if string(answers.Framework) != test.wantFramework || string(answers.Payment) != test.wantPayment || answers.API != test.wantAPI {
				t.Fatalf("answers = %#v", answers)
			}
			if len(answers.Content) != 0 {
				t.Fatalf("content = %#v", answers.Content)
			}
		})
	}
}

func TestNewRejectsInvalidHeadlessSelectionsBeforeGenerating(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{name: "headless with htmx", arguments: []string{"--headless", "--framework", "htmx"}, want: "--headless cannot be combined with --framework"},
		{name: "headless with framework headless", arguments: []string{"--headless", "--framework", "headless"}, want: "--headless cannot be combined with --framework"},
		{name: "headless with content", arguments: []string{"--headless", "--content", "blog"}, want: "Headless does not support blog or docs"},
		{name: "headless with framework svelte", arguments: []string{"--headless", "--framework", "svelte"}, want: "--headless cannot be combined with --framework"},
		{name: "svelte with content", arguments: []string{"--framework", "svelte", "--content", "docs"}, want: "SvelteKit does not support blog or docs"},
		{name: "unknown framework", arguments: []string{"--framework", "react"}, want: `unsupported framework "react"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := testApp(&bytes.Buffer{})
			arguments := append([]string{"new", "--module", "example.com/acme"}, test.arguments...)
			arguments = append(arguments, filepath.Join(t.TempDir(), "acme"))

			err := app.Run(context.Background(), arguments)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("new error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateSelectionRequiresTheJSONAPIForHeadless(t *testing.T) {
	answers := generator.Config{Framework: "headless", Database: "sqlite", Payment: "stripe", Mail: "smtp"}
	err := validateSelection(answers)
	if err == nil || err.Error() != "Headless requires the JSON API" {
		t.Fatalf("validation error = %v", err)
	}
	answers.API = true
	err = validateSelection(answers)
	if err != nil {
		t.Fatalf("validation error = %v", err)
	}
}

func TestValidateSelectionRequiresTheJSONAPIForSvelteKit(t *testing.T) {
	answers := generator.Config{Framework: "svelte", Database: "sqlite", Payment: "polar", Mail: "smtp"}
	err := validateSelection(answers)
	if err == nil || err.Error() != "SvelteKit requires the JSON API" {
		t.Fatalf("validation error = %v", err)
	}
	answers.API = true
	answers.MCP = true
	err = validateSelection(answers)
	if err != nil {
		t.Fatalf("validation error = %v", err)
	}
}

func TestNewHelpFlagShowsHelpNew(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		output := &bytes.Buffer{}
		app := testApp(output)
		err := app.Run(context.Background(), []string{"new", flag})
		if err != nil {
			t.Fatalf("new %s error = %v", flag, err)
		}
		if !strings.HasPrefix(output.String(), "goilerplate new --module <path> [options] <directory>\n") {
			t.Fatalf("new %s output = %q", flag, output.String())
		}
	}
}

func TestHelpNewListsTheFrontendsAndTheJSONAPI(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	err := app.Run(context.Background(), []string{"help", "new"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "edition") || strings.Contains(output.String(), "Free") {
		t.Fatalf("output = %q mentions the removed edition", output.String())
	}
	for _, want := range []string{"No account and no network needed", "--payment stripe or polar", "--framework htmx, datastar, svelte or headless", "--framework svelte", "Node.js 22.17 or newer and pnpm", "--api", "--headless", "--mcp"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output = %q, want %q", output.String(), want)
		}
	}
}

func TestNewSendsTheMCPServer(t *testing.T) {
	for _, test := range []struct {
		name          string
		arguments     []string
		wantFramework string
		wantMCP       bool
	}{
		{name: "htmx with the JSON API", arguments: []string{"--api", "--mcp"}, wantFramework: "htmx", wantMCP: true},
		{name: "datastar with the JSON API", arguments: []string{"--framework", "datastar", "--api", "--mcp"}, wantFramework: "datastar", wantMCP: true},
		{name: "headless", arguments: []string{"--payment", "polar", "--headless", "--mcp"}, wantFramework: "headless", wantMCP: true},
		{name: "htmx with the JSON API only", arguments: []string{"--api"}, wantFramework: "htmx"},
	} {
		t.Run(test.name, func(t *testing.T) {
			answers := runNew(t, test.arguments)
			if string(answers.Framework) != test.wantFramework || !answers.API || answers.MCP != test.wantMCP {
				t.Fatalf("answers = %#v", answers)
			}
		})
	}
}

// runNew runs goilerplate new and returns the answers in the lock file of the
// generated project.
func runNew(t *testing.T, arguments []string) generator.Config {
	t.Helper()
	app := testApp(&bytes.Buffer{})
	destination := filepath.Join(t.TempDir(), "acme")
	command := append([]string{"new", "--module", "example.com/acme"}, arguments...)
	command = append(command, destination)

	err := app.Run(context.Background(), command)
	if err != nil {
		t.Fatalf("new error = %v", err)
	}
	_, answers := readProjectLock(t, destination)
	return answers
}

func readProjectLock(t *testing.T, root string) (string, generator.Config) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "goilerplate.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		TemplateVersion string           `json:"template_version"`
		Config          generator.Config `json:"config"`
	}
	if err := json.Unmarshal(content, &lock); err != nil {
		t.Fatal(err)
	}
	return lock.TemplateVersion, lock.Config
}

func readCLITree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		tree[filepath.ToSlash(name)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}
func TestNewRejectsTheMCPServerWithoutTheJSONAPI(t *testing.T) {
	for _, arguments := range [][]string{
		{"--mcp"},
		{"--framework", "datastar", "--mcp"},
		{"--payment", "polar", "--mcp"},
	} {
		app := testApp(&bytes.Buffer{})
		command := append([]string{"new", "--module", "example.com/acme"}, arguments...)
		command = append(command, filepath.Join(t.TempDir(), "acme"))

		err := app.Run(context.Background(), command)
		if err == nil || err.Error() != "MCP requires the JSON API" {
			t.Fatalf("new %v error = %v", arguments, err)
		}
	}
}

func TestUpdateMergesTheNewTemplateLocally(t *testing.T) {
	oldFiles := templateWith(t, map[string]string{"AGENTS.md.tmpl": "old agents\n"})
	oldTemplate, err := generator.NewTemplate("v4.0.0", oldFiles)
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	answers := generator.Config{ProjectName: "Acme", ModulePath: "example.com/acme", Framework: "headless", Database: "sqlite", Payment: "stripe", Mail: "smtp", API: true, MCP: true}
	if err := oldTemplate.WriteArchive(&archive, answers); err != nil {
		t.Fatal(err)
	}
	repository := filepath.Join(t.TempDir(), "acme")
	if err := project.Create(context.Background(), &archive, repository); err != nil {
		t.Fatal(err)
	}
	runCLIGit(t, repository, "config", "user.name", "Test User")
	runCLIGit(t, repository, "config", "user.email", "test@example.com")
	readme := filepath.Join(repository, "README.md")
	content, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readme, append(content, "Our own notes.\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	runCLIGit(t, repository, "commit", "-am", "Our notes")

	output := &bytes.Buffer{}
	app := testApp(output)
	app.Version = "v4.1.0"
	app.WorkingDirectory = repository
	var fetched []string
	app.FetchTemplate = func(_ context.Context, version string) (fs.FS, error) {
		fetched = append(fetched, version)
		return oldFiles, nil
	}

	if err := app.Run(context.Background(), []string{"update"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(fetched, ",") != "v4.0.0" {
		t.Fatalf("fetched = %v", fetched)
	}
	branch := "goilerplate-update-v4.1.0"
	if agents := runCLIGit(t, repository, "show", branch+":AGENTS.md"); !strings.HasPrefix(agents, "# AGENTS.md\n") {
		t.Fatalf("AGENTS.md on %s = %q", branch, agents)
	}
	if updated := runCLIGit(t, repository, "show", branch+":README.md"); !strings.HasSuffix(updated, "Our own notes.\n") {
		t.Fatalf("README.md lost the project's own change: %q", updated)
	}
	if lock := runCLIGit(t, repository, "show", branch+":goilerplate.lock"); !strings.Contains(lock, `"template_version": "v4.1.0"`) {
		t.Fatalf("goilerplate.lock = %q", lock)
	}
	if current := strings.TrimSpace(runCLIGit(t, repository, "branch", "--show-current")); current != "main" {
		t.Fatalf("current branch = %q", current)
	}
	if !strings.Contains(output.String(), "Created "+branch) {
		t.Fatalf("output = %q", output.String())
	}
}

func TestUpdateAlreadyOnTheCurrentTemplate(t *testing.T) {
	repository := lockedRepository(t, "v4.1.0")
	output := &bytes.Buffer{}
	app := testApp(output)
	app.Version = "v4.1.0"
	app.WorkingDirectory = repository
	app.FetchTemplate = func(context.Context, string) (fs.FS, error) {
		t.Fatal("fetched a template for a project on the current version")
		return nil, nil
	}
	if err := app.Run(context.Background(), []string{"update"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Already using goilerplate template v4.1.0") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestUpdateExplainsWhatItCannotUpdate(t *testing.T) {
	for _, test := range []struct {
		project string
		want    string
	}{
		{project: "v3.2.3", want: "made with goilerplate v3.2.3. goilerplate update works for projects made with v4 or newer. The project keeps working as it is"},
		{project: "v4.0.0-dev", want: "development build"},
		{project: "v4.2.0", want: "newer than this CLI (v4.1.0)"},
		{project: "latest", want: "invalid template version"},
	} {
		t.Run(test.project, func(t *testing.T) {
			app := testApp(&bytes.Buffer{})
			app.Version = "v4.1.0"
			app.WorkingDirectory = lockedRepository(t, test.project)
			app.FetchTemplate = func(context.Context, string) (fs.FS, error) {
				t.Fatal("fetched a template for a project it cannot update")
				return nil, nil
			}
			err := app.Run(context.Background(), []string{"update"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("update error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestUpdateNamesAVersionMissingFromTheProxy(t *testing.T) {
	app := testApp(&bytes.Buffer{})
	app.Version = "v4.1.0"
	app.WorkingDirectory = lockedRepository(t, "v4.0.0")
	app.FetchTemplate = func(_ context.Context, version string) (fs.FS, error) {
		return nil, fmt.Errorf("%w: %s", projectupdate.ErrVersionNotFound, version)
	}
	err := app.Run(context.Background(), []string{"update"})
	if err == nil || !strings.Contains(err.Error(), "goilerplate v4.0.0 was not found on the Go module proxy") {
		t.Fatalf("update error = %v", err)
	}
}

// lockedRepository creates a Git repository whose goilerplate.lock names version.
func lockedRepository(t *testing.T, version string) string {
	t.Helper()
	config := map[string]any{
		"project_name": "Acme", "module_path": "example.com/acme",
		"framework": "htmx", "database": "sqlite", "payment": "stripe", "mail": "smtp",
	}
	if strings.HasPrefix(version, "v3.") {
		// goilerplate v3 wrote the edition into every lock file.
		config["edition"] = "paid"
	}
	lock, err := json.MarshalIndent(map[string]any{
		"schema_version":   generator.LockSchemaVersion,
		"template_version": version,
		"config":           config,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	repository := t.TempDir()
	writeCLIFiles(t, repository, map[string]string{"goilerplate.lock": string(lock) + "\n"})
	runCLIGit(t, repository, "init", "-b", "main")
	runCLIGit(t, repository, "config", "user.name", "Test User")
	runCLIGit(t, repository, "config", "user.email", "test@example.com")
	runCLIGit(t, repository, "add", ".")
	runCLIGit(t, repository, "commit", "-m", "Generated project")
	return repository
}

// templateWith returns the template of this checkout with some files replaced.
func templateWith(t *testing.T, replaced map[string]string) fstest.MapFS {
	t.Helper()
	files := make(fstest.MapFS)
	source := os.DirFS(filepath.Join("..", "..", "generator"))
	err := fs.WalkDir(source, "_template", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		files[name] = &fstest.MapFile{Data: data, Mode: 0o644}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range replaced {
		name = "_template/" + name
		if _, ok := files[name]; !ok {
			t.Fatalf("template has no %s", name)
		}
		files[name] = &fstest.MapFile{Data: []byte(content), Mode: 0o644}
	}
	return files
}

func testApp(output *bytes.Buffer) *App {
	return &App{Out: output, Version: "dev"}
}

func writeCLIFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runCLIGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func TestHelpNewListsEveryOption(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output)
	err := app.Run(context.Background(), []string{"help", "new"})
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range []string{"--module", "--name", "--framework", "--database", "--payment", "--mail", "--workspaces", "--oauth", "--storage", "--content", "--api", "--mcp", "--headless"} {
		if !strings.Contains(output.String(), option) {
			t.Errorf("help new does not mention %s", option)
		}
	}
}
