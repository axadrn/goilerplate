package generator

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/mod/module"
)

type Framework string
type Database string
type Payment string
type Mail string
type OAuth string
type Content string

const (
	FrameworkHTMX     Framework = "htmx"
	FrameworkDatastar Framework = "datastar"
	FrameworkHeadless Framework = "headless"
	FrameworkSvelte   Framework = "svelte"

	DatabaseSQLite   Database = "sqlite"
	DatabasePostgres Database = "postgres"

	PaymentPolar  Payment = "polar"
	PaymentStripe Payment = "stripe"

	MailSMTP   Mail = "smtp"
	MailResend Mail = "resend"

	OAuthGoogle OAuth = "google"
	OAuthGitHub OAuth = "github"

	ContentBlog Content = "blog"
	ContentDocs Content = "docs"
)

type Config struct {
	ProjectName string    `json:"project_name"`
	ModulePath  string    `json:"module_path"`
	Framework   Framework `json:"framework"`
	Database    Database  `json:"database"`
	Payment     Payment   `json:"payment"`
	Mail        Mail      `json:"mail"`
	Workspaces  bool      `json:"workspaces"`
	OAuth       []OAuth   `json:"oauth"`
	Storage     bool      `json:"storage"`
	Content     []Content `json:"content"`
	API         bool      `json:"api"`
	MCP         bool      `json:"mcp"`
}

func (c Config) IsHeadless() bool { return c.Framework == FrameworkHeadless }

func (c Config) IsSvelte() bool { return c.Framework == FrameworkSvelte }

// HasHTMLFrontend reports whether the Go server renders HTML pages with templ.
// Headless and SvelteKit builds serve only the JSON API from Go.
func (c Config) HasHTMLFrontend() bool {
	return c.Framework == FrameworkHTMX || c.Framework == FrameworkDatastar
}

func (c Config) HasGoogleOAuth() bool { return slices.Contains(c.OAuth, OAuthGoogle) }

func (c Config) HasGitHubOAuth() bool { return slices.Contains(c.OAuth, OAuthGitHub) }

func (c Config) HasBlog() bool { return slices.Contains(c.Content, ContentBlog) }

func (c Config) HasDocs() bool { return slices.Contains(c.Content, ContentDocs) }

func (c Config) DatabaseTimestamp() string {
	if c.Database == DatabasePostgres {
		return "TIMESTAMPTZ"
	}
	return "TIMESTAMP"
}

func (c Config) ProjectNameGo() string { return strconv.Quote(c.ProjectName) }

func (c Config) YAMLProjectTeamName() string {
	return strconv.Quote(c.ProjectName + " Team")
}

func (c Config) normalized() (Config, error) {
	c.ProjectName = strings.TrimSpace(c.ProjectName)
	c.ModulePath = strings.TrimSpace(c.ModulePath)
	c.OAuth = uniqueOAuth(c.OAuth)
	c.Content = uniqueContent(c.Content)

	var validationErrors []error
	if c.ProjectName == "" {
		validationErrors = append(validationErrors, errors.New("project name is required"))
	}
	if strings.IndexFunc(c.ProjectName, unicode.IsControl) >= 0 {
		validationErrors = append(validationErrors, errors.New("project name cannot contain control characters"))
	}
	if err := validateModulePath(c.ModulePath); err != nil {
		validationErrors = append(validationErrors, err)
	}
	if c.Framework != FrameworkHTMX && c.Framework != FrameworkDatastar && c.Framework != FrameworkHeadless && c.Framework != FrameworkSvelte {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported framework %q", c.Framework))
	}
	if c.Database != DatabaseSQLite && c.Database != DatabasePostgres {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported database %q", c.Database))
	}
	if c.Payment == "" {
		validationErrors = append(validationErrors, errors.New("a payment provider is required"))
	} else if c.Payment != PaymentPolar && c.Payment != PaymentStripe {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported payment provider %q", c.Payment))
	}
	if c.Mail != MailSMTP && c.Mail != MailResend {
		validationErrors = append(validationErrors, fmt.Errorf("unsupported mail provider %q", c.Mail))
	}
	for _, provider := range c.OAuth {
		if provider != OAuthGoogle && provider != OAuthGitHub {
			validationErrors = append(validationErrors, fmt.Errorf("unsupported OAuth provider %q", provider))
		}
	}
	for _, feature := range c.Content {
		if feature != ContentBlog && feature != ContentDocs {
			validationErrors = append(validationErrors, fmt.Errorf("unsupported content feature %q", feature))
		}
	}
	if c.IsHeadless() && !c.API {
		validationErrors = append(validationErrors, errors.New("Headless requires the JSON API"))
	}
	if c.IsHeadless() && len(c.Content) != 0 {
		validationErrors = append(validationErrors, errors.New("Headless does not support blog or docs"))
	}
	if c.IsSvelte() && !c.API {
		validationErrors = append(validationErrors, errors.New("SvelteKit requires the JSON API"))
	}
	if c.IsSvelte() && len(c.Content) != 0 {
		validationErrors = append(validationErrors, errors.New("SvelteKit does not support blog or docs"))
	}
	if c.MCP && !c.API {
		validationErrors = append(validationErrors, errors.New("MCP requires the JSON API"))
	}
	if len(validationErrors) > 0 {
		return Config{}, errors.Join(validationErrors...)
	}
	return c, nil
}

func validateModulePath(modulePath string) error {
	if modulePath == "" {
		return errors.New("module path is required")
	}
	if err := module.CheckPath(modulePath); err != nil {
		return fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}
	return nil
}

func uniqueOAuth(values []OAuth) []OAuth {
	result := make([]OAuth, 0, len(values))
	for _, candidate := range []OAuth{OAuthGoogle, OAuthGitHub} {
		if slices.Contains(values, candidate) {
			result = append(result, candidate)
		}
	}
	for _, value := range values {
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func uniqueContent(values []Content) []Content {
	result := make([]Content, 0, len(values))
	for _, candidate := range []Content{ContentBlog, ContentDocs} {
		if slices.Contains(values, candidate) {
			result = append(result, candidate)
		}
	}
	for _, value := range values {
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}
