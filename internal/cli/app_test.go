package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axadrn/goilerplate/api"
	"github.com/axadrn/goilerplate/internal/config"
	"github.com/axadrn/goilerplate/internal/github"
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
	if !strings.Contains(output.String(), "* paid  active  member") {
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
	if service.generateToken != "session" || service.generateRequest.Answers.Payment != "stripe" || !service.generateRequest.Answers.Teams {
		t.Fatalf("generation request = %#v", service.generateRequest)
	}
	if !strings.Contains(output.String(), "Created Acme") {
		t.Fatalf("output = %q", output.String())
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
	login               api.GitHubLoginResponse
	who                 api.WhoAmIResponse
	receivedGitHubToken string
	loggedOut           bool
	logoutError         error
	generateRequest     api.GenerateRequest
	generateToken       string
	generatedVersion    string
	archive             []byte
	generateCalled      bool
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

func cliTestArchive(t *testing.T, name, content string) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
