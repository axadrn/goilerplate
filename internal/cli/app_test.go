package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/config"
	"github.com/axadrn/goilerplate/v3/internal/doctor"
	"github.com/axadrn/goilerplate/v3/internal/github"
)

func TestLoginStoresOnlyGoilerplateSession(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{}
	device := &fakeDevice{githubToken: "temporary-github-token"}
	service := &fakeService{
		login: api.GitHubLoginResponse{
			SessionToken: "goilerplate-session",
			Account:      api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
		},
	}
	app := testApp(output, store, device, service)

	if err := app.Run(context.Background(), []string{"login"}); err != nil {
		t.Fatal(err)
	}
	if service.receivedGitHubToken != "temporary-github-token" {
		t.Fatalf("GitHub token = %q", service.receivedGitHubToken)
	}
	if store.configuration.SessionToken != "goilerplate-session" {
		t.Fatalf("stored session = %q", store.configuration.SessionToken)
	}
	if strings.Contains(output.String(), "temporary-github-token") || strings.Contains(output.String(), "goilerplate-session") {
		t.Fatalf("secret appeared in output: %q", output.String())
	}
}

func TestLoginPrefersAndStoresAPIURLOverride(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com"}}
	service := &fakeService{login: api.GitHubLoginResponse{
		SessionToken: "beta-session",
		Account:      api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
	}}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{githubToken: "temporary-github-token"}, service)
	app.APIURLOverride = "https://beta.goilerplate.com"
	var serviceURL string
	app.NewService = func(baseURL string) (ServiceClient, error) {
		serviceURL = baseURL
		return service, nil
	}

	if err := app.Run(context.Background(), []string{"login"}); err != nil {
		t.Fatal(err)
	}
	if serviceURL != "https://beta.goilerplate.com" {
		t.Fatalf("service URL = %q", serviceURL)
	}
	if store.configuration.APIURL != "https://beta.goilerplate.com" {
		t.Fatalf("stored API URL = %q", store.configuration.APIURL)
	}
}

func TestSignedInCommandPrefersAPIURLOverride(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{who: api.WhoAmIResponse{Account: api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"}}}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)
	app.APIURLOverride = "https://beta.goilerplate.com"
	var serviceURL string
	app.NewService = func(baseURL string) (ServiceClient, error) {
		serviceURL = baseURL
		return service, nil
	}

	if err := app.Run(context.Background(), []string{"whoami"}); err != nil {
		t.Fatal(err)
	}
	if serviceURL != "https://beta.goilerplate.com" {
		t.Fatalf("service URL = %q", serviceURL)
	}
}

func TestLoginWaitsForFreeActivation(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{}
	service := &fakeService{
		login: api.GitHubLoginResponse{
			SessionToken:       "goilerplate-session",
			Account:            api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
			ActivationRequired: true,
		},
		activationActiveAfter: 2,
	}
	app := testApp(output, store, &fakeDevice{githubToken: "temporary-github-token"}, service)
	app.ActivationPollInterval = time.Millisecond

	if err := app.Run(context.Background(), []string{"login"}); err != nil {
		t.Fatal(err)
	}
	if service.activationChecks != 2 {
		t.Fatalf("activation checks = %d, want 2", service.activationChecks)
	}
	if !strings.Contains(output.String(), "Free access activated") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestLoginRevokesNewSessionWhenSavingFails(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{saveError: errors.New("disk full")}
	service := &fakeService{login: api.GitHubLoginResponse{
		SessionToken: "goilerplate-session",
		Account:      api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
	}}
	app := testApp(output, store, &fakeDevice{githubToken: "temporary-github-token"}, service)

	if err := app.Run(context.Background(), []string{"login"}); !errors.Is(err, store.saveError) {
		t.Fatalf("login error = %v", err)
	}
	if !service.loggedOut {
		t.Fatal("new server session was not revoked")
	}
	if store.configuration.SessionToken != "" {
		t.Fatalf("stored session = %q", store.configuration.SessionToken)
	}
}

func TestWhoAmIShowsEffectiveLicense(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{who: api.WhoAmIResponse{
		Account: api.Account{GitHubLogin: "axadrn", Email: "hello@example.com"},
		Licenses: []api.License{
			{ID: "free", Tier: api.LicenseTierFree, Status: api.LicenseStatusActive, Role: api.LicenseRoleOwner},
			{ID: "paid", Tier: api.LicenseTierPaid, Status: api.LicenseStatusActive, Role: api.LicenseRoleMember},
		},
		EffectiveLicenseID: "paid",
	}}
	app := testApp(output, store, &fakeDevice{}, service)

	if err := app.Run(context.Background(), []string{"whoami"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "* paid  paid  active  member") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestLogoutRevokesServerBeforeClearingSession(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)
	if err := app.Run(context.Background(), []string{"logout"}); err != nil {
		t.Fatal(err)
	}
	if store.configuration.SessionToken != "" || store.configuration.APIURL == "" {
		t.Fatalf("configuration = %#v", store.configuration)
	}
	if !service.loggedOut {
		t.Fatal("server session was not revoked")
	}
}

func TestLogoutPreservesTokenWhenRevocationFails(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{logoutError: errors.New("temporary failure")}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)
	if err := app.Run(context.Background(), []string{"logout"}); !errors.Is(err, service.logoutError) {
		t.Fatalf("logout error = %v", err)
	}
	if store.configuration.SessionToken != "session" {
		t.Fatalf("session token = %q, want preserved token", store.configuration.SessionToken)
	}
}

func TestLicenseAndTokenCommandsUseExplicitLicenseIDs(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{
		members: api.LicenseMembersResponse{Members: []api.LicenseMember{{
			UserID: "user-1", GitHubLogin: "developer", Email: "developer@example.com", Role: api.LicenseRoleMember,
		}}},
		createdToken: api.CreateLicenseTokenResponse{
			Token: api.LicenseToken{ID: "token-1", Name: "deploy"}, Value: "gok_secret",
		},
	}
	app := testApp(output, store, &fakeDevice{}, service)
	if err := app.Run(context.Background(), []string{"license", "members", "license-1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(context.Background(), []string{"token", "create", "license-1", "deploy"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "@developer") || !strings.Contains(output.String(), "gok_secret") || !strings.Contains(output.String(), "shown only once") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestAccountDeleteRequiresConfirmationAndClearsLocalSession(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)
	if err := app.Run(context.Background(), []string{"account", "delete"}); err == nil {
		t.Fatal("account delete accepted no confirmation")
	}
	if err := app.Run(context.Background(), []string{"account", "delete", "--confirm", "axadrn"}); err != nil {
		t.Fatal(err)
	}
	if service.deletedAccount != "axadrn" || store.configuration.SessionToken != "" {
		t.Fatalf("confirmation = %q, configuration = %#v", service.deletedAccount, store.configuration)
	}
}

func TestClaimStartsAndConfirmsPurchaseEmailProof(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{}
	app := testApp(output, store, &fakeDevice{}, service)

	if err := app.Run(context.Background(), []string{"claim", "buyer@example.com"}); err != nil {
		t.Fatal(err)
	}
	if service.claimEmail != "buyer@example.com" || !strings.Contains(output.String(), "claim code") {
		t.Fatalf("email = %q, output = %q", service.claimEmail, output.String())
	}
	if err := app.Run(context.Background(), []string{"claim", "--code", "abcd2345ef"}); err != nil {
		t.Fatal(err)
	}
	if service.claimCode != "ABCD2345EF" || !strings.Contains(output.String(), "License claimed") {
		t.Fatalf("code = %q, output = %q", service.claimCode, output.String())
	}
}

func TestClaimValidatesArgumentsBeforeLogin(t *testing.T) {
	app := testApp(&bytes.Buffer{}, &memoryStore{}, &fakeDevice{}, &fakeService{})
	if err := app.Run(context.Background(), []string{"claim"}); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("claim error = %v", err)
	}
}

func TestHelpCoversEveryCommand(t *testing.T) {
	for _, command := range []string{"new", "update", "login", "whoami", "logout", "activation", "claim", "license", "token", "account", "changelog", "doctor", "version"} {
		t.Run(command, func(t *testing.T) {
			output := &bytes.Buffer{}
			app := testApp(output, &memoryStore{}, &fakeDevice{}, &fakeService{})
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
	app := testApp(output, &memoryStore{}, &fakeDevice{}, &fakeService{})
	if err := app.Run(context.Background(), []string{"wat"}); err == nil {
		t.Fatal("unknown command succeeded")
	}
	if !strings.Contains(output.String(), "Usage: goilerplate <command>") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestChangelogPrintsPublishedReleaseNotes(t *testing.T) {
	output := &bytes.Buffer{}
	app := testApp(output, &memoryStore{}, &fakeDevice{}, &fakeService{})
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
	app := testApp(output, &memoryStore{}, &fakeDevice{}, &fakeService{})
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

func TestNewGeneratesAndExtractsProject(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{generatedVersion: "v3.0.0", archive: cliTestArchive(t, "go.mod", "module example.com/acme")}
	app := testApp(output, store, &fakeDevice{}, service)
	destination := filepath.Join(t.TempDir(), "acme")

	err := app.Run(context.Background(), []string{
		"new",
		"--name", "Acme",
		"--module", "example.com/acme",
		"--edition", "paid",
		"--framework", "datastar",
		"--database", "postgres",
		"--teams",
		"--oauth", "google,github",
		destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "go.mod"))
	if err != nil || string(content) != "module example.com/acme" {
		t.Fatalf("go.mod = %q, %v", content, err)
	}
	if service.generateToken != "session" || service.generateRequest.Answers.Framework != "datastar" || service.generateRequest.Answers.Payment != "stripe" || !service.generateRequest.Answers.Teams {
		t.Fatalf("generation request = %#v", service.generateRequest)
	}
	if !strings.Contains(output.String(), "Created Acme") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestNewWithoutArgumentsUsesWizardThenExistingGenerationPath(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{generatedVersion: "v3.0.0", archive: cliTestArchive(t, "go.mod", "module example.com/acme")}
	app := testApp(output, store, &fakeDevice{}, service)
	destination := filepath.Join(t.TempDir(), "acme")
	wizardCalled := false
	app.RunNewProjectWizard = func(context.Context) ([]string, error) {
		wizardCalled = true
		return []string{
			"--name", "Acme",
			"--module", "example.com/acme",
			"--edition", "paid",
			"--database", "postgres",
			"--teams",
			destination,
		}, nil
	}

	if err := app.Run(context.Background(), []string{"new"}); err != nil {
		t.Fatal(err)
	}
	if !wizardCalled || !service.generateCalled {
		t.Fatalf("wizard called = %v, generated = %v", wizardCalled, service.generateCalled)
	}
	if service.generateRequest.Answers.Database != "postgres" || !service.generateRequest.Answers.Teams {
		t.Fatalf("generation request = %#v", service.generateRequest)
	}
}

func TestNewReturnsWizardErrorWithoutCallingService(t *testing.T) {
	wizardError := errors.New("cancelled")
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, &memoryStore{}, &fakeDevice{}, service)
	app.RunNewProjectWizard = func(context.Context) ([]string, error) {
		return nil, wizardError
	}

	if err := app.Run(context.Background(), []string{"new"}); !errors.Is(err, wizardError) {
		t.Fatalf("new error = %v", err)
	}
	if service.generateCalled {
		t.Fatal("service was called after wizard failure")
	}
}

func TestNewWithoutArgumentsRequiresInteractiveTerminal(t *testing.T) {
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, &memoryStore{}, &fakeDevice{}, service)

	err := app.Run(context.Background(), []string{"new"})
	if err == nil || !strings.Contains(err.Error(), "usage: goilerplate new") {
		t.Fatalf("new error = %v", err)
	}
	if service.generateCalled {
		t.Fatal("service was called without project arguments")
	}
}

func TestNewUsesMachineTokenWithoutPersonalLoginOrActivation(t *testing.T) {
	store := &memoryStore{}
	service := &fakeService{generatedVersion: "v3.0.0", archive: cliTestArchive(t, "go.mod", "module example.com/acme")}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)
	app.MachineToken = "gok_machine"
	if err := app.Run(context.Background(), []string{
		"new", "--module", "example.com/acme", filepath.Join(t.TempDir(), "acme"),
	}); err != nil {
		t.Fatal(err)
	}
	if service.generateToken != "gok_machine" || service.activationChecks != 0 || service.receivedGitHubToken != "" {
		t.Fatalf("generate token = %q, activation checks = %d, GitHub token = %q", service.generateToken, service.activationChecks, service.receivedGitHubToken)
	}
}

func TestNewRejectsPaidModulesForFreeBeforeCallingService(t *testing.T) {
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, &memoryStore{}, &fakeDevice{}, service)
	err := app.Run(context.Background(), []string{
		"new", "--module", "example.com/acme", "--teams", filepath.Join(t.TempDir(), "acme"),
	})
	if err == nil || !strings.Contains(err.Error(), "Free uses SQLite") {
		t.Fatalf("new error = %v", err)
	}
	if service.generateCalled {
		t.Fatal("service was called for an invalid selection")
	}
}

func TestNewRejectsDatastarForFreeBeforeCallingService(t *testing.T) {
	service := &fakeService{}
	app := testApp(&bytes.Buffer{}, &memoryStore{}, &fakeDevice{}, service)
	err := app.Run(context.Background(), []string{
		"new", "--module", "example.com/acme", "--framework", "datastar", filepath.Join(t.TempDir(), "acme"),
	})
	if err == nil || !strings.Contains(err.Error(), "Free uses SQLite") {
		t.Fatalf("new error = %v", err)
	}
	if service.generateCalled {
		t.Fatal("service was called for an invalid Free frontend")
	}
}

func TestUpdateCreatesGitBranchFromLockedAnswers(t *testing.T) {
	lock := api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Answers: api.GenerationAnswers{
			ProjectName: "Acme", ModulePath: "example.com/acme", Edition: "paid",
			Framework: "htmx", Database: "sqlite", Payment: "stripe", Mail: "smtp",
		},
	}
	lockBytes, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	lockBytes = append(lockBytes, '\n')
	oldFiles := map[string]string{"app.txt": "old\n", "goilerplate.lock": string(lockBytes)}
	repository := t.TempDir()
	writeCLIFiles(t, repository, oldFiles)
	runCLIGit(t, repository, "init", "-b", "main")
	runCLIGit(t, repository, "config", "user.name", "Test User")
	runCLIGit(t, repository, "config", "user.email", "test@example.com")
	runCLIGit(t, repository, "add", ".")
	runCLIGit(t, repository, "commit", "-m", "Generated project")

	newLock := lock
	newLock.TemplateVersion = "v3.1.0"
	newLockBytes, err := json.MarshalIndent(newLock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	newLockBytes = append(newLockBytes, '\n')
	service := &fakeService{
		oldUpdateArchive: cliTestArchiveFiles(t, oldFiles),
		newUpdateArchive: cliTestArchiveFiles(t, map[string]string{"app.txt": "new\n", "goilerplate.lock": string(newLockBytes)}),
		updateVersion:    "v3.1.0",
	}
	output := &bytes.Buffer{}
	app := testApp(output, &memoryStore{configuration: config.Config{SessionToken: "session"}}, &fakeDevice{}, service)
	app.WorkingDirectory = repository

	if err := app.Run(context.Background(), []string{"update"}); err != nil {
		t.Fatal(err)
	}
	if len(service.updateRequests) != 2 || service.updateRequests[0].TemplateVersion != "v3.0.0" || service.updateRequests[1].TemplateVersion != "" {
		t.Fatalf("update requests = %#v", service.updateRequests)
	}
	if content := runCLIGit(t, repository, "show", "goilerplate-update-v3.1.0:app.txt"); content != "new\n" {
		t.Fatalf("updated app = %q", content)
	}
	if branch := strings.TrimSpace(runCLIGit(t, repository, "branch", "--show-current")); branch != "main" {
		t.Fatalf("current branch = %q", branch)
	}
	if !strings.Contains(output.String(), "Created goilerplate-update-v3.1.0") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestNewResumesPendingFreeActivation(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{
		activationActiveAfter: 2,
		generatedVersion:      "v3.0.0",
		archive:               cliTestArchive(t, "go.mod", "module example.com/acme"),
	}
	app := testApp(output, store, &fakeDevice{}, service)
	app.ActivationPollInterval = time.Millisecond
	destination := filepath.Join(t.TempDir(), "acme")

	if err := app.Run(context.Background(), []string{"new", "--module", "example.com/acme", destination}); err != nil {
		t.Fatal(err)
	}
	if service.activationChecks != 2 || !service.generateCalled {
		t.Fatalf("activation checks = %d, generated = %v", service.activationChecks, service.generateCalled)
	}
}

func TestNewDoesNotWaitWithoutPendingActivation(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{activationUnavailable: true}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)

	err := app.Run(context.Background(), []string{"new", "--module", "example.com/acme", filepath.Join(t.TempDir(), "acme")})
	if !errors.Is(err, errActivationUnavailable) {
		t.Fatalf("new error = %v, want activation guidance", err)
	}
	if service.activationChecks != 1 || service.generateCalled {
		t.Fatalf("activation checks = %d, generated = %v", service.activationChecks, service.generateCalled)
	}
}

func TestNewGuidesExpiredActivationToResend(t *testing.T) {
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{activationResendRequired: true}
	app := testApp(&bytes.Buffer{}, store, &fakeDevice{}, service)

	err := app.Run(context.Background(), []string{"new", "--module", "example.com/acme", filepath.Join(t.TempDir(), "acme")})
	if !errors.Is(err, errActivationResendRequired) {
		t.Fatalf("new error = %v, want resend guidance", err)
	}
}

func TestActivationResendSendsAndWaits(t *testing.T) {
	output := &bytes.Buffer{}
	store := &memoryStore{configuration: config.Config{APIURL: "https://goilerplate.com", SessionToken: "session"}}
	service := &fakeService{activationUnavailable: true, activationActiveAfter: 2}
	app := testApp(output, store, &fakeDevice{}, service)
	app.ActivationPollInterval = time.Millisecond

	if err := app.Run(context.Background(), []string{"activation", "resend"}); err != nil {
		t.Fatal(err)
	}
	if !service.activationResent || service.activationChecks != 2 {
		t.Fatalf("resent = %v, activation checks = %d", service.activationResent, service.activationChecks)
	}
	if !strings.Contains(output.String(), "Activation email sent") || !strings.Contains(output.String(), "Free access activated") {
		t.Fatalf("output = %q", output.String())
	}
}

func testApp(output *bytes.Buffer, store *memoryStore, device *fakeDevice, service *fakeService) *App {
	return &App{
		Out:            output,
		Store:          store,
		Device:         device,
		GitHubClientID: "client-123",
		DefaultAPIURL:  "https://goilerplate.com",
		Version:        "dev",
		NewService: func(string) (ServiceClient, error) {
			return service, nil
		},
	}
}

type memoryStore struct {
	configuration config.Config
	saveError     error
}

func (s *memoryStore) Load() (config.Config, error) {
	return s.configuration, nil
}

func (s *memoryStore) Save(configuration config.Config) error {
	if s.saveError != nil {
		return s.saveError
	}
	s.configuration = configuration
	return nil
}

type fakeDevice struct {
	githubToken string
}

func (d *fakeDevice) Start(context.Context, string) (github.DeviceAuthorization, error) {
	return github.DeviceAuthorization{UserCode: "ABCD-EFGH", VerificationURI: "https://github.com/login/device"}, nil
}

func (d *fakeDevice) Wait(context.Context, string, github.DeviceAuthorization) (string, error) {
	return d.githubToken, nil
}

type fakeService struct {
	login                    api.GitHubLoginResponse
	who                      api.WhoAmIResponse
	receivedGitHubToken      string
	loggedOut                bool
	logoutError              error
	generateRequest          api.GenerateRequest
	generateToken            string
	generatedVersion         string
	archive                  []byte
	oldUpdateArchive         []byte
	newUpdateArchive         []byte
	updateVersion            string
	updateRequests           []api.GenerateRequest
	generateCalled           bool
	activationChecks         int
	activationActiveAfter    int
	activationUnavailable    bool
	activationResendRequired bool
	activationResent         bool
	members                  api.LicenseMembersResponse
	tokens                   api.LicenseTokensResponse
	createdToken             api.CreateLicenseTokenResponse
	invited                  api.InviteLicenseMemberResponse
	removedMember            string
	revokedToken             string
	deletedAccount           string
	claimEmail               string
	claimCode                string
}

func (s *fakeService) LoginWithGitHub(_ context.Context, token string) (api.GitHubLoginResponse, error) {
	s.receivedGitHubToken = token
	return s.login, nil
}

func (s *fakeService) WhoAmI(context.Context, string) (api.WhoAmIResponse, error) {
	return s.who, nil
}

func (s *fakeService) Logout(context.Context, string) error {
	s.loggedOut = true
	return s.logoutError
}

func (s *fakeService) Generate(_ context.Context, token string, request api.GenerateRequest, destination io.Writer) (string, error) {
	s.generateCalled = true
	s.generateToken = token
	s.generateRequest = request
	if _, err := destination.Write(s.archive); err != nil {
		return "", err
	}
	return s.generatedVersion, nil
}

func (s *fakeService) UpdateTree(_ context.Context, token string, request api.GenerateRequest, destination io.Writer) (string, error) {
	s.generateToken = token
	s.updateRequests = append(s.updateRequests, request)
	archive := s.newUpdateArchive
	version := s.updateVersion
	if request.TemplateVersion != "" {
		archive = s.oldUpdateArchive
		version = request.TemplateVersion
	}
	if _, err := destination.Write(archive); err != nil {
		return "", err
	}
	return version, nil
}

func (s *fakeService) ActivationStatus(context.Context, string) (api.ActivationStatusResponse, error) {
	s.activationChecks++
	state := api.ActivationStatePending
	if s.activationUnavailable {
		state = api.ActivationStateUnavailable
	} else if s.activationResendRequired {
		state = api.ActivationStateResendRequired
	} else if s.activationActiveAfter == 0 || s.activationChecks >= s.activationActiveAfter {
		state = api.ActivationStateActive
	}
	return api.ActivationStatusResponse{State: state}, nil
}

func (s *fakeService) ResendActivation(context.Context, string) (api.ActivationStatusResponse, error) {
	s.activationResent = true
	s.activationUnavailable = false
	s.activationResendRequired = false
	return api.ActivationStatusResponse{State: api.ActivationStatePending}, nil
}

func (s *fakeService) BeginLicenseClaim(_ context.Context, _ string, email string) error {
	s.claimEmail = email
	return nil
}

func (s *fakeService) ConfirmLicenseClaim(_ context.Context, _ string, code string) error {
	s.claimCode = code
	return nil
}

func (s *fakeService) LicenseMembers(context.Context, string, string) (api.LicenseMembersResponse, error) {
	return s.members, nil
}

func (s *fakeService) InviteLicenseMember(context.Context, string, string, api.InviteLicenseMemberRequest) (api.InviteLicenseMemberResponse, error) {
	return s.invited, nil
}

func (s *fakeService) RemoveLicenseMember(_ context.Context, _ string, _ string, userID string) error {
	s.removedMember = userID
	return nil
}

func (s *fakeService) CreateLicenseToken(context.Context, string, string, string) (api.CreateLicenseTokenResponse, error) {
	return s.createdToken, nil
}

func (s *fakeService) LicenseTokens(context.Context, string, string) (api.LicenseTokensResponse, error) {
	return s.tokens, nil
}

func (s *fakeService) RevokeLicenseToken(_ context.Context, _ string, _ string, tokenID string) error {
	s.revokedToken = tokenID
	return nil
}

func (s *fakeService) DeleteAccount(_ context.Context, _ string, confirmation string) error {
	s.deletedAccount = confirmation
	return nil
}

func cliTestArchive(t *testing.T, name, content string) []byte {
	return cliTestArchiveFiles(t, map[string]string{name: content})
}

func cliTestArchiveFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
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
