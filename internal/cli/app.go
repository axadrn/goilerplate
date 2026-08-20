package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/api"
	"github.com/axadrn/goilerplate/internal/config"
	"github.com/axadrn/goilerplate/internal/github"
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
	ActivationStatus(context.Context, string) (api.ActivationStatusResponse, error)
	Generate(context.Context, string, api.GenerateRequest, io.Writer) (string, error)
}

type App struct {
	Out                    io.Writer
	Store                  ConfigStore
	Device                 DeviceAuthorizer
	NewService             func(string) (ServiceClient, error)
	GitHubClientID         string
	DefaultAPIURL          string
	Version                string
	ActivationPollInterval time.Duration
}

func (a *App) Run(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 {
		a.printUsage()
		return nil
	}
	switch arguments[0] {
	case "new":
		return a.newProject(ctx, arguments[1:])
	case "login":
		return a.login(ctx, arguments[1:])
	case "whoami":
		return a.whoAmI(ctx, arguments[1:])
	case "logout":
		return a.logout(ctx, arguments[1:])
	case "version":
		return a.version(arguments[1:])
	case "help", "-h", "--help":
		a.printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
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
	apiURL := configuration.APIURL
	if apiURL == "" {
		apiURL = a.DefaultAPIURL
	}
	client, err := a.NewService(apiURL)
	if err != nil {
		return err
	}
	authorization, err := a.Device.Start(ctx, a.GitHubClientID)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Open %s and enter code %s\n", authorization.VerificationURI, authorization.UserCode)
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
	if login.ActivationRequired {
		fmt.Fprintln(a.Out, "Check your email to activate Free access. Waiting for confirmation...")
		if err := a.waitForActivation(ctx, client, login.SessionToken); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, "Free access activated")
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
	apiURL := configuration.APIURL
	if apiURL == "" {
		apiURL = a.DefaultAPIURL
	}
	client, err := a.NewService(apiURL)
	if err != nil {
		return err
	}
	identity, err := client.WhoAmI(ctx, configuration.SessionToken)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "@%s <%s>\n", identity.Account.GitHubLogin, identity.Account.Email)
	for _, license := range identity.Licenses {
		marker := " "
		if license.ID == identity.EffectiveLicenseID {
			marker = "*"
		}
		fmt.Fprintf(a.Out, "%s %s  %s  %s\n", marker, license.Tier, license.Status, license.Role)
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
	apiURL := configuration.APIURL
	if apiURL == "" {
		apiURL = a.DefaultAPIURL
	}
	client, err := a.NewService(apiURL)
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
	fmt.Fprintln(a.Out, "Goilerplate")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Usage: goilerplate <command>")
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Commands:")
	fmt.Fprintln(a.Out, "  new      Generate a new project")
	fmt.Fprintln(a.Out, "  login    Sign in with GitHub")
	fmt.Fprintln(a.Out, "  whoami   Show the current account and licenses")
	fmt.Fprintln(a.Out, "  logout   Revoke the current session")
	fmt.Fprintln(a.Out, "  version  Show the CLI version")
}
