package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/config"
	"github.com/axadrn/goilerplate/v3/internal/doctor"
	"github.com/axadrn/goilerplate/v3/internal/github"
)

type ConfigStore interface {
	Load() (config.Config, error)
	Save(config.Config) error
}

type DeviceAuthorizer interface {
	Start(context.Context, string) (github.DeviceAuthorization, error)
	Wait(context.Context, string, github.DeviceAuthorization) (string, error)
}

type ServiceClient interface {
	LoginWithGitHub(context.Context, string) (api.GitHubLoginResponse, error)
	WhoAmI(context.Context, string) (api.WhoAmIResponse, error)
	Logout(context.Context, string) error
	BeginLicenseClaim(context.Context, string, string) error
	ConfirmLicenseClaim(context.Context, string, string) error
	LicenseMembers(context.Context, string, string) (api.LicenseMembersResponse, error)
	InviteLicenseMember(context.Context, string, string, api.InviteLicenseMemberRequest) (api.InviteLicenseMemberResponse, error)
	RemoveLicenseMember(context.Context, string, string, string) error
	DeleteAccount(context.Context, string, string) error
	Generate(context.Context, string, api.GenerateRequest, io.Writer) (string, error)
	UpdateTree(context.Context, string, api.GenerateRequest, io.Writer) (string, error)
}

type App struct {
	Out                 io.Writer
	Store               ConfigStore
	Device              DeviceAuthorizer
	OpenBrowser         func(context.Context, string) error
	CopyToClipboard     func(context.Context, string) error
	NewService          func(string) (ServiceClient, error)
	GitHubClientID      string
	DefaultAPIURL       string
	APIURLOverride      string
	Version             string
	WorkingDirectory    string
	RunNewProjectWizard func(context.Context) ([]string, error)
	FetchReleases       func(context.Context) ([]github.Release, error)
	RunDoctor           func(context.Context, string) doctor.Report
}

func (a *App) apiURL(configuration config.Config) string {
	if apiURL := strings.TrimSpace(a.APIURLOverride); apiURL != "" {
		return apiURL
	}
	if apiURL := strings.TrimSpace(configuration.APIURL); apiURL != "" {
		return apiURL
	}
	return strings.TrimSpace(a.DefaultAPIURL)
}

func (a *App) signedInClient() (ServiceClient, string, error) {
	configuration, err := a.Store.Load()
	if err != nil {
		return nil, "", err
	}
	if configuration.SessionToken == "" {
		return nil, "", errors.New("not signed in. Run goilerplate login")
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return nil, "", err
	}
	return client, configuration.SessionToken, nil
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
	case "login":
		return a.login(ctx, arguments[1:])
	case "whoami":
		return a.whoAmI(ctx, arguments[1:])
	case "logout":
		return a.logout(ctx, arguments[1:])
	case "claim":
		return a.claim(ctx, arguments[1:])
	case "license":
		return a.license(ctx, arguments[1:])
	case "account":
		return a.account(ctx, arguments[1:])
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
		"new":       {"goilerplate new [options] <directory>", "Run without options in a terminal to open interactive setup."},
		"update":    {"goilerplate update", "Create a Git branch containing the new generated template."},
		"login":     {"goilerplate login", "Sign in through GitHub's device flow."},
		"whoami":    {"goilerplate whoami", "Show the current account and available licenses."},
		"logout":    {"goilerplate logout", "Revoke the current goilerplate session."},
		"claim":     {"goilerplate claim <purchase-email>", "goilerplate claim --code <code>"},
		"license":   {"goilerplate license members|invite|remove ...", "Manage people on a company license."},
		"account":   {"goilerplate account delete --confirm <github-login>", "Delete the current account."},
		"changelog": {"goilerplate changelog", "Show published GitHub release notes."},
		"doctor":    {"goilerplate doctor", "Check tools and configuration for a generated project."},
		"version":   {"goilerplate version", "Show the CLI version."},
	}
	lines, ok := usage[arguments[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", arguments[0])
	}
	for _, line := range lines {
		fmt.Fprintln(a.Out, line)
	}
	return nil
}

func (a *App) login(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate login")
	}
	if strings.TrimSpace(a.GitHubClientID) == "" {
		return errors.New("GitHub login is not configured")
	}
	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	apiURL := a.apiURL(configuration)
	client, err := a.NewService(apiURL)
	if err != nil {
		return err
	}
	authorization, err := a.Device.Start(ctx, a.GitHubClientID)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "GitHub login")
	fmt.Fprintln(a.Out)
	copied := a.CopyToClipboard != nil && a.CopyToClipboard(ctx, authorization.UserCode) == nil
	if copied {
		fmt.Fprintf(a.Out, "Code: %s (copied)\n", authorization.UserCode)
	} else {
		fmt.Fprintf(a.Out, "Code: %s\n", authorization.UserCode)
	}
	fmt.Fprintf(a.Out, "Open: %s\n", authorization.VerificationURI)
	fmt.Fprintln(a.Out)
	if a.OpenBrowser != nil && a.OpenBrowser(ctx, authorization.VerificationURI) == nil {
		fmt.Fprintln(a.Out, "Browser opened. Waiting for approval...")
	} else {
		fmt.Fprintln(a.Out, "Waiting for approval...")
	}
	githubToken, err := a.Device.Wait(ctx, a.GitHubClientID, authorization)
	if err != nil {
		return err
	}
	login, err := client.LoginWithGitHub(ctx, githubToken)
	if err != nil {
		return err
	}
	configuration.APIURL = apiURL
	configuration.SessionToken = login.SessionToken
	if err := a.Store.Save(configuration); err != nil {
		_ = client.Logout(ctx, login.SessionToken)
		return err
	}
	fmt.Fprintf(a.Out, "Signed in as @%s\n", login.Account.GitHubLogin)
	return nil
}

func (a *App) whoAmI(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate whoami")
	}
	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	if configuration.SessionToken == "" {
		return errors.New("not signed in. Run goilerplate login")
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return err
	}
	identity, err := client.WhoAmI(ctx, configuration.SessionToken)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "@%s <%s>\n", identity.Account.GitHubLogin, identity.Account.Email)
	access := "Free"
	for _, license := range identity.Licenses {
		if license.Status == api.LicenseStatusActive {
			access = "Paid"
			break
		}
	}
	fmt.Fprintf(a.Out, "Access: %s\n", access)
	if len(identity.Licenses) == 0 {
		return nil
	}
	fmt.Fprintln(a.Out, "  LICENSE ID  STATUS  ROLE")
	for _, license := range identity.Licenses {
		fmt.Fprintf(a.Out, "  %s  %s  %s\n", license.ID, license.Status, license.Role)
	}
	return nil
}

func (a *App) logout(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate logout")
	}
	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	if configuration.SessionToken == "" {
		fmt.Fprintln(a.Out, "Already signed out")
		return nil
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return err
	}
	if err := client.Logout(ctx, configuration.SessionToken); err != nil {
		return err
	}
	configuration.SessionToken = ""
	if err := a.Store.Save(configuration); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "Signed out")
	return nil
}

func (a *App) version(arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate version")
	}
	fmt.Fprintln(a.Out, a.Version)
	return nil
}

func (a *App) printUsage() {
	fmt.Fprintln(a.Out, "goilerplate")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Usage: goilerplate <command>")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Commands:")
	fmt.Fprintln(a.Out, "  new      Generate a new project")
	fmt.Fprintln(a.Out, "  update   Update a generated project on a new Git branch")
	fmt.Fprintln(a.Out, "  login    Sign in with GitHub")
	fmt.Fprintln(a.Out, "  claim    Connect a purchase made with another email")
	fmt.Fprintln(a.Out, "  whoami   Show the current account and licenses")
	fmt.Fprintln(a.Out, "  license  Invite, list, or remove license members")
	fmt.Fprintln(a.Out, "  account delete  Delete the current account")
	fmt.Fprintln(a.Out, "  changelog  Show published release notes")
	fmt.Fprintln(a.Out, "  doctor   Check a generated project's local tools")
	fmt.Fprintln(a.Out, "  logout   Revoke the current session")
	fmt.Fprintln(a.Out, "  version  Show the CLI version")
}
