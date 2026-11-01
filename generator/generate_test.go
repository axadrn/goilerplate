package generator

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBrandAssetsMatchTheSite(t *testing.T) {
	for _, name := range []string{"favicon.svg", "og.png"} {
		t.Run(name, func(t *testing.T) {
			siteAsset, err := os.ReadFile(filepath.Join("..", "site", "assets", "img", name))
			if err != nil {
				t.Fatalf("read site asset: %v", err)
			}
			templateAsset, err := os.ReadFile(filepath.Join("_template", "assets.htmx.datastar", "img", name))
			if err != nil {
				t.Fatalf("read template asset: %v", err)
			}
			if !bytes.Equal(templateAsset, siteAsset) {
				t.Fatalf("generator %s differs from the site asset", name)
			}
		})
	}
}

func TestGeneratedProjectShipsWithoutALicenseFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", fullSQLiteHTMXConfig()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"} {
		if _, err := os.Stat(filepath.Join(destination, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("generated project contains %s: %v", name, err)
		}
	}
	readme, err := os.ReadFile(filepath.Join(destination, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "0BSD") || strings.Contains(string(readme), "(LICENSE)") {
		t.Fatalf("README.md does not explain the license of the generated project")
	}
}

func TestGeneratedProjectIncludesAutomaticDevelopmentPorts(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", fullSQLiteHTMXConfig()); err != nil {
		t.Fatal(err)
	}
	tree := readTree(t, destination)

	if _, ok := tree["cmd/devport/main.go"]; !ok {
		t.Fatal("generated project does not contain the development port helper")
	}
	taskfile := string(tree["Taskfile.yml"])
	for _, expected := range []string{
		"go run ./cmd/devport",
		"--proxyport=\"{{.DEV_PORT}}\"",
		"--open-browser=\"{{.OPEN_BROWSER}}\"",
		"APP_URL: 'http://localhost:{{.DEV_PORT}}'",
		"Local: http://localhost:{{.DEV_PORT}}",
	} {
		if !strings.Contains(taskfile, expected) {
			t.Errorf("generated Taskfile does not contain %q", expected)
		}
	}
}

func TestGeneratedProjectLinksToDocumentation(t *testing.T) {
	config := Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkHTMX,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
	}
	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", config); err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join(destination, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(readme, []byte("goilerplate.com/docs")) {
		t.Error("generated README does not link to the public documentation")
	}
}

func TestGeneratedDocsAreLLMReadableWithoutExposingAgentInstructions(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", fullSQLiteHTMXConfig()); err != nil {
		t.Fatal(err)
	}
	tree := readTree(t, destination)

	llms := string(tree["content/llms.txt"])
	if strings.Contains(llms, "AGENTS.md") || strings.Contains(llms, "internal/app") {
		t.Fatalf("public llms.txt exposes repository instructions: %s", llms)
	}
	if !strings.Contains(string(tree["internal/routes/routes.go"]), `mux.HandleFunc("GET /docs.md", docs.ShowDocs)`) {
		t.Fatal("generated routes do not serve root documentation as Markdown")
	}
	seoHandler := string(tree["internal/handler/seo.go"])
	for _, expected := range []string{"FlatDocsList()", `page.Slug + ".md"`, "## Documentation"} {
		if !strings.Contains(seoHandler, expected) {
			t.Fatalf("generated LLM index does not contain %q", expected)
		}
	}
	layout := string(tree["internal/ui/layouts/base.templ"])
	for _, expected := range []string{`rel="describedby"`, `rel="alternate"`, `type="text/markdown"`} {
		if !strings.Contains(layout, expected) {
			t.Fatalf("generated layout does not contain %q", expected)
		}
	}
	docs := string(tree["content/docs/getting-started.md"])
	if !strings.Contains(docs, "```bash") || strings.Contains(docs, "goilerplate documentation") {
		t.Fatal("generated docs are not a generic, code-backed product example")
	}
}

func TestGeneratedProjectsUseSelectedMailProviderInEveryEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		provider string
	}{
		{name: "minimal htmx SMTP", config: minimalConfig(), provider: "NewSMTPMailer"},
		{name: "full htmx SMTP", config: fullSQLiteHTMXConfig(), provider: "NewSMTPMailer"},
		{
			name: "full htmx Resend",
			config: func() Config {
				config := fullSQLiteHTMXConfig()
				config.Mail = MailResend
				return config
			}(),
			provider: "NewResendMailer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "project")
			if err := Generate(destination, "3.0.0", test.config); err != nil {
				t.Fatal(err)
			}
			tree := readTree(t, destination)
			runtimeContract := string(tree[".env.example"]) + string(tree["internal/config/config.go"])
			for _, forbidden := range []string{"DB_DRIVER", "MAIL_PROVIDER"} {
				if strings.Contains(runtimeContract, forbidden) {
					t.Errorf("runtime contract contains generated architecture key %s", forbidden)
				}
			}
			if test.config.Mail == MailSMTP && strings.Contains(runtimeContract, "ResendAPIKey") {
				t.Error("SMTP config contains Resend configuration")
			}
			if test.config.Mail == MailResend && strings.Contains(runtimeContract, "SMTPHost") {
				t.Error("Resend config contains SMTP configuration")
			}
			for _, path := range []string{
				"internal/service/mailer/console.go",
				"internal/service/mailer/console_test.go",
			} {
				if _, ok := tree[path]; ok {
					t.Errorf("generated project contains %s", path)
				}
			}

			factory := string(tree["internal/service/mailer/factory.go"])
			if !strings.Contains(factory, test.provider) {
				t.Errorf("mailer factory does not construct %s", test.provider)
			}
			if strings.Contains(factory, "IsDevelopment") || strings.Contains(factory, "ConsoleMailer") {
				t.Fatal("mailer factory contains a development-only provider path")
			}

			for path, content := range tree {
				if filepath.Ext(path) != ".go" {
					continue
				}
				for lineNumber, line := range strings.Split(string(content), "\n") {
					if !strings.Contains(line, "slog.") {
						continue
					}
					for _, sensitive := range []string{`"email"`, `"new_email"`, `"old_email"`, `"to"`, `"token"`, `"name"`} {
						if strings.Contains(line, sensitive) {
							t.Errorf("%s:%d logs sensitive field %s", path, lineNumber+1, sensitive)
						}
					}
				}
			}
		})
	}
}

func TestGenerateIsByteDeterministic(t *testing.T) {
	const templateVersion = "test-version"
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	config := fullSQLiteHTMXConfig()
	config.OAuth = []OAuth{OAuthGitHub, OAuthGoogle, OAuthGitHub}
	config.Content = []Content{ContentDocs, ContentBlog, ContentDocs}
	if err := Generate(first, templateVersion, config); err != nil {
		t.Fatalf("first Generate() error = %v", err)
	}
	if err := Generate(second, templateVersion, config); err != nil {
		t.Fatalf("second Generate() error = %v", err)
	}
	firstTree := readTree(t, first)
	secondTree := readTree(t, second)
	if !reflect.DeepEqual(firstTree, secondTree) {
		t.Fatal("identical generation inputs produced different trees")
	}
	if !strings.HasPrefix(string(firstTree["go.mod"]), "module "+config.ModulePath+"\n") {
		t.Fatalf("generated go.mod has wrong module path: %q", firstTree["go.mod"])
	}
	var lockFields map[string]json.RawMessage
	if err := json.Unmarshal(firstTree["goilerplate.lock"], &lockFields); err != nil {
		t.Fatalf("decode lock fields: %v", err)
	}
	if _, ok := lockFields["config"]; !ok {
		t.Fatal("generated lock has no config field")
	}
	if _, ok := lockFields["answers"]; ok {
		t.Fatal("generated lock contains answers field")
	}

	var lock LockFile
	if err := json.Unmarshal(firstTree["goilerplate.lock"], &lock); err != nil {
		t.Fatalf("decode lock: %v", err)
	}
	if lock.SchemaVersion != LockSchemaVersion || lock.TemplateVersion != templateVersion {
		t.Fatalf("lock metadata = %#v", lock)
	}
	if !reflect.DeepEqual(lock.Config.OAuth, []OAuth{OAuthGoogle, OAuthGitHub}) || !reflect.DeepEqual(lock.Config.Content, []Content{ContentBlog, ContentDocs}) {
		t.Fatalf("normalized lock config = %#v", lock.Config)
	}
}

func TestGenerateRejectsNonEmptyDestination(t *testing.T) {
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(destination, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(destination, "3.0.0", fullSQLiteHTMXConfig()); err == nil || !strings.Contains(err.Error(), "destination must be empty") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestSpecRejectsUnsafeModulePaths(t *testing.T) {
	for _, modulePath := range []string{
		`github.com/example/bad"module`,
		"github.com/example/bad#module",
		"github.com/example/bad:module",
		"github.com/example/bad module",
		"github.com/example/.hidden",
		"github.com/example/name.",
		"github.com/example/CON",
	} {
		config := fullSQLiteHTMXConfig()
		config.ModulePath = modulePath
		if _, err := config.normalized(); err == nil || !strings.Contains(err.Error(), "invalid module path") {
			t.Errorf("module path %q error = %v", modulePath, err)
		}
	}
}

func TestGenerateUsesProjectNameSafelyAndConsistently(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.ProjectName = `Rocket "$Lab" #1`
	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", config); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	tree := readTree(t, destination)
	if strings.Contains(string(tree[".env.example"]), "APP_NAME=") {
		t.Fatalf(".env contains source-owned project name: %q", tree[".env.example"])
	}
	if !strings.Contains(string(tree["internal/brand/brand.go"]), `Name        = "Rocket \"$Lab\" #1"`) {
		t.Fatal("brand package does not use the project name safely")
	}
	blogPosts := []string{
		"building-on-a-solid-saas-foundation.md",
		"choosing-a-billing-model-that-scales.md",
		"designing-your-first-customer-workflow.md",
		"launch-checklist-for-a-reliable-saas.md",
	}
	for _, post := range blogPosts {
		path := filepath.Join("content", "blog", post)
		if !strings.Contains(string(tree[path]), `author: "Rocket \"$Lab\" #1 Team"`) {
			t.Errorf("%s author does not use the project name", path)
		}
	}
}

func TestOlderTemplatesRejectTheJSONAPI(t *testing.T) {
	older := fstest.MapFS{
		"_template/README.md":                   {Data: []byte("older\n")},
		"_template/internal/handler/home.go":    {Data: []byte("package handler\n")},
		"_template/internal/ui/base.htmx.templ": {Data: []byte("package ui\n")},
	}
	template, err := NewTemplate("v3.0.3", older)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkHTMX,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
	}
	if err := template.Generate(t.TempDir(), config); err != nil {
		t.Fatalf("older template without the API error = %v", err)
	}
	for _, framework := range []Framework{FrameworkHTMX, FrameworkHeadless} {
		config.Framework = framework
		config.API = true
		err := template.Generate(t.TempDir(), config)
		if !errors.Is(err, ErrUnsupportedForVersion) || !strings.Contains(err.Error(), APIFirstVersion) {
			t.Fatalf("older template with %s and the API error = %v, want ErrUnsupportedForVersion naming %s", framework, err, APIFirstVersion)
		}
	}

	older["_template/internal/api/root.api.go"] = &fstest.MapFile{Data: []byte("package api\n")}
	config.Framework = FrameworkHeadless
	if err := template.Generate(t.TempDir(), config); err != nil {
		t.Fatalf("template with API sources error = %v", err)
	}
	current, err := CurrentTemplate("v3.1.0")
	if err != nil {
		t.Fatal(err)
	}
	supported, err := current.hasAxis("api")
	if err != nil || !supported {
		t.Fatalf("current template hasAxis(api) = %v, %v", supported, err)
	}
}

func TestOlderTemplatesRejectTheMCPServer(t *testing.T) {
	older := fstest.MapFS{
		"_template/README.md":                   {Data: []byte("older\n")},
		"_template/internal/api/root.api.go":    {Data: []byte("package api\n")},
		"_template/internal/ui/base.htmx.templ": {Data: []byte("package ui\n")},
	}
	template, err := NewTemplate("v3.1.0-test", older)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkHTMX,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
		API:         true,
	}
	err = template.Generate(t.TempDir(), config)
	if err != nil {
		t.Fatalf("template without MCP sources and the API error = %v", err)
	}
	config.MCP = true
	err = template.Generate(t.TempDir(), config)
	var versionError UnsupportedForVersionError
	if !errors.Is(err, ErrUnsupportedForVersion) || !errors.As(err, &versionError) {
		t.Fatalf("template without MCP sources error = %v, want ErrUnsupportedForVersion", err)
	}
	want := "The MCP server needs goilerplate " + MCPFirstVersion + " or newer"
	if versionError.Message != want {
		t.Fatalf("message = %q, want %q", versionError.Message, want)
	}

	older["_template/internal/mcp/server.mcp.go"] = &fstest.MapFile{Data: []byte("package mcp\n")}
	err = template.Generate(t.TempDir(), config)
	if err != nil {
		t.Fatalf("template with MCP sources error = %v", err)
	}
	config.API = false
	err = template.Generate(t.TempDir(), config)
	if !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), "MCP requires the JSON API") {
		t.Fatalf("MCP without the API error = %v, want ErrInvalidConfig", err)
	}
}

func TestOlderTemplatesRejectSvelteKit(t *testing.T) {
	config := Config{
		ProjectName: "Acme",
		ModulePath:  "github.com/example/acme",
		Framework:   FrameworkSvelte,
		Database:    DatabaseSQLite,
		Payment:     PaymentStripe,
		Mail:        MailSMTP,
		API:         true,
	}
	want := "SvelteKit needs goilerplate " + SvelteFirstVersion + " or newer"
	for name, older := range map[string]fstest.MapFS{
		"before the API": {
			"_template/README.md":                   {Data: []byte("older\n")},
			"_template/internal/ui/base.htmx.templ": {Data: []byte("package ui\n")},
		},
		"with Headless": {
			"_template/README.md":                     {Data: []byte("older\n")},
			"_template/internal/api/root.headless.go": {Data: []byte("package api\n")},
			"_template/internal/ui/base.htmx.templ":   {Data: []byte("package ui\n")},
		},
	} {
		template, err := NewTemplate("v3.0.3", older)
		if err != nil {
			t.Fatal(err)
		}
		err = template.Generate(t.TempDir(), config)
		var versionError UnsupportedForVersionError
		if !errors.Is(err, ErrUnsupportedForVersion) || !errors.As(err, &versionError) || versionError.Message != want {
			t.Fatalf("%s: SvelteKit error = %v, want %q", name, err, want)
		}
	}

	withSvelte := fstest.MapFS{
		"_template/README.md":                            {Data: []byte("newer\n")},
		"_template/internal/api/root.headless.svelte.go": {Data: []byte("package api\n")},
		"_template/internal/api/auth.api.go":             {Data: []byte("package api\n")},
		"_template/web.svelte/svelte.config.js":          {Data: []byte("export default {}\n")},
	}
	template, err := NewTemplate("v3.1.0", withSvelte)
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	err = template.Generate(destination, config)
	if err != nil {
		t.Fatalf("template with SvelteKit sources error = %v", err)
	}
	for _, path := range []string{"internal/api/root.go", "internal/api/auth.go", "web/svelte.config.js"} {
		_, err := os.Stat(filepath.Join(destination, path))
		if err != nil {
			t.Fatalf("SvelteKit output is missing %s: %v", path, err)
		}
	}
	config.API = false
	err = template.Generate(t.TempDir(), config)
	if !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), "SvelteKit requires the JSON API") {
		t.Fatalf("SvelteKit without the API error = %v, want ErrInvalidConfig", err)
	}
	config.API = true
	config.Content = []Content{ContentBlog}
	err = template.Generate(t.TempDir(), config)
	if !errors.Is(err, ErrInvalidConfig) || !strings.Contains(err.Error(), "SvelteKit does not support blog or docs") {
		t.Fatalf("SvelteKit with the blog error = %v, want ErrInvalidConfig", err)
	}

	current, err := CurrentTemplate("v3.1.0")
	if err != nil {
		t.Fatal(err)
	}
	supported, err := current.hasWord("svelte")
	if err != nil || !supported {
		t.Fatalf("current template hasWord(svelte) = %v, %v", supported, err)
	}
}

func TestGeneratedFrontendContainsExactlyOneFramework(t *testing.T) {
	for _, framework := range []Framework{FrameworkHTMX, FrameworkDatastar} {
		config := fullSQLiteHTMXConfig()
		config.Framework = framework
		destination := filepath.Join(t.TempDir(), string(framework))
		if err := Generate(destination, "3.0.0", config); err != nil {
			t.Fatalf("Generate(%s) error = %v", framework, err)
		}
		tree := readTree(t, destination)
		nativeHandler := "internal/handler/" + string(framework) + ".go"
		if _, ok := tree[nativeHandler]; !ok {
			t.Fatalf("%s output is missing %s", framework, nativeHandler)
		}
		combined := string(tree["go.mod"]) + string(tree[nativeHandler]) + string(tree["internal/middleware/security_headers.go"]) + string(tree["internal/middleware/frontend_policy.go"])
		routesSource := string(tree["internal/routes/routes.go"])
		if !strings.Contains(routesSource, "public, max-age=31536000, immutable") {
			t.Errorf("%s output does not cache content-hashed JavaScript as immutable", framework)
		}
		var frontendSource strings.Builder
		for path := range tree {
			if path == "internal/ui/respond.go" || path == "internal/ui/respond_frontend.go" || strings.HasPrefix(path, "internal/ui/frontend/") {
				t.Errorf("%s output contains obsolete frontend abstraction %s", framework, path)
			}
			switch filepath.Ext(path) {
			case ".go", ".templ", ".css":
				frontendSource.Write(tree[path])
			}
		}

		if framework == FrameworkDatastar {
			if !strings.Contains(frontendSource.String(), `src="/assets/js/datastar-1.0.3.5d6b7794.js"`) {
				t.Error("Datastar output does not reference its immutable browser runtime path")
			}
			for _, path := range []string{"assets/js/datastar-1.0.3.5d6b7794.js", "assets/js/datastar-license.txt"} {
				if _, ok := tree[path]; !ok {
					t.Fatalf("Datastar output is missing %s", path)
				}
			}
			for _, path := range []string{"assets/js/htmx-4.0.0.e484d917.min.js", "assets/js/htmx-license.txt"} {
				if _, ok := tree[path]; ok {
					t.Fatalf("Datastar output contains %s", path)
				}
			}
			for _, required := range []string{"github.com/starfederation/datastar-go", "Datastar-Request", ".Redirect(location)", "datastar.WithSelector(selector)", "datastar.WithModeAppend()"} {
				if !strings.Contains(combined, required) {
					t.Errorf("Datastar output is missing %q", required)
				}
			}
			if strings.Contains(combined, "'unsafe-eval'") {
				t.Error("Datastar output contains unsafe-eval instead of CSP nonce mode")
			}
			if !strings.Contains(frontendSource.String(), "data-nonce") {
				t.Error("Datastar output is missing the CSP nonce marker")
			}
			if !strings.Contains(frontendSource.String(), `form[aria-busy] button[type="submit"]`) {
				t.Error("Datastar output does not show loading state for boolean aria-busy attributes")
			}
			for _, forbidden := range []string{"hx-post=", "hx-patch=", "hx-delete=", "HX-Request", `data-on:submit="(`, "goilerplateRedirect", "datastar-signal-patch", "renderOOB", "beforeend:#"} {
				if strings.Contains(frontendSource.String(), forbidden) {
					t.Errorf("Datastar output contains %q", forbidden)
				}
			}
			continue
		}

		if !strings.Contains(frontendSource.String(), `src="/assets/js/htmx-4.0.0.e484d917.min.js"`) {
			t.Error("htmx output does not reference its immutable browser runtime path")
		}
		for _, path := range []string{"assets/js/htmx-4.0.0.e484d917.min.js", "assets/js/htmx-license.txt"} {
			if _, ok := tree[path]; !ok {
				t.Fatalf("htmx output is missing %s", path)
			}
		}
		for _, path := range []string{"assets/js/datastar-1.0.3.5d6b7794.js", "assets/js/datastar-license.txt"} {
			if _, ok := tree[path]; ok {
				t.Fatalf("htmx output contains %s", path)
			}
		}
		for _, forbidden := range []string{"github.com/starfederation/datastar-go", "Datastar-Request", "'unsafe-eval'"} {
			if strings.Contains(combined, forbidden) {
				t.Errorf("htmx output contains %q", forbidden)
			}
		}
		for _, forbidden := range []string{"data-on:", "data-indicator", "datastar.NewSSE", "goilerplate-loading"} {
			if strings.Contains(frontendSource.String(), forbidden) {
				t.Errorf("htmx output contains %q", forbidden)
			}
		}
		for _, required := range []string{"hx-headers:inherited", "hx-disable=", "htmx:after:settle", `hx-include="closest form"`} {
			if !strings.Contains(frontendSource.String(), required) {
				t.Errorf("htmx 4.0 output is missing %q", required)
			}
		}
		for _, forbidden := range []string{"hx-disabled-elt", "htmx:afterSettle", "htmx:afterSwap", "hx-headers="} {
			if strings.Contains(frontendSource.String(), forbidden) {
				t.Errorf("htmx 4.0 output contains legacy token %q", forbidden)
			}
		}
	}
}

func TestGeneratedSamePageMutationComponentsArePresent(t *testing.T) {
	for _, framework := range []Framework{FrameworkHTMX, FrameworkDatastar} {
		for _, docs := range []bool{false, true} {
			config := fullSQLiteHTMXConfig()
			config.Framework = framework
			config.Content = nil
			if docs {
				config.Content = []Content{ContentDocs}
			}
			destination := filepath.Join(t.TempDir(), fmt.Sprintf("%s-docs-%t", framework, docs))
			if err := Generate(destination, "3.0.0", config); err != nil {
				t.Fatalf("Generate(%s, docs=%t) error = %v", framework, docs, err)
			}
			tree := readTree(t, destination)
			for path, required := range map[string]string{
				"internal/handler/admin.go":                    "pages.AdminDashboardContent",
				"internal/handler/workspace.go":                "layouts.WorkspaceIdentity",
				"internal/handler/settings.go":                 "pages.SettingsSessionsSection",
				"internal/ui/layouts/app_identity.templ":       "@WorkspaceIdentity(ctxkeys.Workspace(ctx), ctxkeys.Workspaces(ctx))",
				"internal/ui/layouts/workspace_identity.templ": "templ WorkspaceIdentity",
				"internal/ui/pages/admin_dashboard.templ":      "templ AdminDashboardContent",
				"internal/ui/pages/app_settings.templ":         `ID: "settings-sessions"`,
				"internal/ui/pages/app_workspace.templ":        "templ WorkspaceContent",
			} {
				if !strings.Contains(string(tree[path]), required) {
					t.Errorf("%s output with docs=%t is missing %q from %s", framework, docs, required, path)
				}
			}
		}
	}
}

func TestVendoredHTMXBundleMatchesV4_0_0(t *testing.T) {
	content, err := os.ReadFile("_template/assets.htmx.datastar/js/htmx-4.0.0.e484d917.min.htmx.js")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	if got, want := fmt.Sprintf("%x", digest), "e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f"; got != want {
		t.Fatalf("htmx bundle SHA-256 = %s, want %s", got, want)
	}
}

func TestVendoredDatastarBundleMatchesV1_0_3(t *testing.T) {
	content, err := os.ReadFile("_template/assets.htmx.datastar/js/datastar-1.0.3.5d6b7794.datastar.js")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	if got, want := fmt.Sprintf("%x", digest), "5d6b7794a50a83d82da962aec5e382f5ae83ac7afbc751f903f7a9c6bd433c65"; got != want {
		t.Fatalf("Datastar bundle SHA-256 = %s, want %s", got, want)
	}
}

func TestGenerateOmitsUnselectedFeatures(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.Workspaces = false
	config.OAuth = nil
	config.Storage = false
	config.Content = nil
	config.Payment = PaymentStripe

	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", config); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	tree := readTree(t, destination)

	for _, path := range []string{
		"internal/service/payment/stripe.go",
		"internal/service/mailer/smtp.go",
		"internal/db/migrations/001_initial_schema.sql",
	} {
		if _, ok := tree[path]; !ok {
			t.Errorf("selected file %s is missing", path)
		}
	}
	for _, path := range []string{
		"internal/handler/auth_google.go",
		"internal/handler/auth_github.go",
		"internal/service/payment/polar.go",
		"internal/service/mailer/resend.go",
		"internal/service/workspace.go",
		"internal/service/file.go",
		"internal/service/blog.go",
		"internal/service/docs.go",
		"internal/service/project.go",
		"internal/db/migrations/002_add_payment_provider.sql",
		"internal/db/migrations/003_add_file_storage.sql",
		"internal/db/migrations/004_add_newsletter.sql",
		"internal/db/migrations/005_add_projects.sql",
		"internal/db/migrations/006_simplify_users_and_tokens.sql",
		"internal/db/migrations/007_rename_organizations_to_workspaces.sql",
		"cmd/seed/main.go",
		"compose.yml",
	} {
		if _, ok := tree[path]; ok {
			t.Errorf("unselected file %s was generated", path)
		}
	}

	combined := string(tree[".env.example"]) + string(tree["go.mod"]) + string(tree["internal/ui/blocks/navbar.templ"]) + string(tree["internal/ui/blocks/footer.templ"]) + string(tree["internal/ui/layouts/app.templ"])
	for _, forbidden := range []string{"GOOGLE_CLIENT_ID", "GITHUB_CLIENT_ID", "S3_BUCKET", "RESEND_API_KEY", "POLAR_API_KEY", "golang.org/x/oauth2", "aws-sdk-go", "resend-go", "polarsource", `href=\"/blog\"`, `href=\"/docs\"`, `\"/app/projects\"`} {
		if strings.Contains(combined, forbidden) {
			t.Errorf("unselected feature marker %q remains", forbidden)
		}
	}
}

func TestGenerateIncludesDerivedPostgresFilesAndSelectedIntegrations(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.Database = DatabasePostgres
	config.Payment = PaymentPolar
	config.Mail = MailResend
	config.OAuth = []OAuth{OAuthGoogle}
	config.Storage = false
	config.Content = []Content{ContentDocs}

	destination := filepath.Join(t.TempDir(), "project")
	if err := Generate(destination, "3.0.0", config); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	tree := readTree(t, destination)
	for _, path := range []string{"compose.yml", "internal/handler/auth_google.go", "internal/service/payment/polar.go", "internal/service/mailer/resend.go", "internal/service/docs.go"} {
		if _, ok := tree[path]; !ok {
			t.Errorf("selected file %s is missing", path)
		}
	}
	for _, path := range []string{"internal/handler/auth_github.go", "internal/service/payment/stripe.go", "internal/service/mailer/smtp.go", "internal/service/file.go", "internal/service/blog.go", "internal/service/project.go"} {
		if _, ok := tree[path]; ok {
			t.Errorf("unselected file %s was generated", path)
		}
	}
	moduleFile := string(tree["go.mod"])
	for _, required := range []string{"github.com/jackc/pgx/v5", "golang.org/x/oauth2", "github.com/resend/resend-go/v2", "github.com/polarsource/polar-go"} {
		if !strings.Contains(moduleFile, required) {
			t.Errorf("go.mod is missing %q", required)
		}
	}
}

func TestGeneratedMigrationsUseDatabaseTimestampType(t *testing.T) {
	for _, test := range []struct {
		name      string
		database  Database
		want      string
		forbidden string
	}{
		{name: "SQLite", database: DatabaseSQLite, want: "TIMESTAMP", forbidden: "TIMESTAMPTZ"},
		{name: "PostgreSQL", database: DatabasePostgres, want: "TIMESTAMPTZ", forbidden: "TIMESTAMP"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := fullSQLiteHTMXConfig()
			config.Database = test.database
			destination := filepath.Join(t.TempDir(), "project")
			if err := Generate(destination, "3.0.0", config); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			tree := readTree(t, destination)
			driver := "sqlite"
			dialect := "sqlite3"
			if test.database == DatabasePostgres {
				driver = "pgx"
				dialect = "postgres"
			}
			databaseSource := string(tree["internal/db/db.go"])
			if !strings.Contains(databaseSource, `databaseDriver  = "`+driver+`"`) {
				t.Errorf("generated app does not fix database driver %q in code", driver)
			}
			if !strings.Contains(databaseSource, `databaseDialect = "`+dialect+`"`) {
				t.Errorf("generated app does not fix database dialect %q in code", dialect)
			}
			if strings.Contains(string(tree["internal/config/config.go"]), "DBDriver") {
				t.Error("generated runtime config contains database driver")
			}
			found := false
			for path, content := range tree {
				if !strings.HasPrefix(path, "internal/db/migrations/") {
					continue
				}
				for _, field := range strings.Fields(string(content)) {
					timestampType := strings.TrimRight(field, ",;")
					if timestampType == test.want {
						found = true
					}
					if timestampType == test.forbidden {
						t.Errorf("%s contains %s", path, test.forbidden)
					}
				}
				if strings.Contains(string(content), "{{.DatabaseTimestamp}}") {
					t.Errorf("%s contains an unresolved timestamp type", path)
				}
			}
			if !found {
				t.Fatalf("generated migrations do not contain %s", test.want)
			}
		})
	}
}

func TestGeneratedTestsMatchSelectedFeatures(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		included []string
		excluded []string
	}{
		{
			name:     "minimal Stripe SQLite",
			config:   minimalConfig(),
			included: []string{"internal/service/payment/factory_test.go", "internal/testutil/database_sqlite.go"},
			excluded: []string{"internal/testpostgres/postgres.go", "internal/handler/auth_google_test.go", "internal/storage/s3_test.go"},
		},
		{
			name:   "Stripe SQLite with selected integrations",
			config: fullSQLiteHTMXConfig(),
			included: []string{
				"internal/service/payment/factory_test.go", "internal/service/payment/stripe_workspace_test.go",
				"internal/handler/auth_google_test.go", "internal/handler/auth_github_test.go", "internal/storage/s3_test.go",
				"internal/testutil/database_sqlite.go",
			},
			excluded: []string{
				"internal/service/payment/polar_workspace_test.go", "internal/testpostgres/postgres.go",
				"internal/testutil/database_postgres.go",
			},
		},
		{
			name: "Polar PostgreSQL",
			config: func() Config {
				config := fullSQLiteHTMXConfig()
				config.Database = DatabasePostgres
				config.Payment = PaymentPolar
				config.OAuth = []OAuth{OAuthGoogle}
				return config
			}(),
			included: []string{
				"internal/testpostgres/postgres.go", "internal/testutil/database_postgres.go",
				"internal/db/postgres_test.go", "internal/repository/postgres_test.go", "internal/service/postgres_test.go",
				"internal/service/payment/postgres_test.go", "internal/service/payment/polar_workspace_test.go",
				"internal/handler/auth_google_test.go",
			},
			excluded: []string{
				"internal/testutil/database_sqlite.go", "internal/service/payment/stripe_workspace_test.go",
				"internal/handler/auth_github_test.go",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "project")
			if err := Generate(destination, "3.0.0", test.config); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			tree := readTree(t, destination)
			for _, path := range test.included {
				if _, ok := tree[path]; !ok {
					t.Errorf("selected test file %s is missing", path)
				}
			}
			for _, path := range test.excluded {
				if _, ok := tree[path]; ok {
					t.Errorf("unselected test file %s was generated", path)
				}
			}
			workflow := string(tree[".github/workflows/ci.yml"])
			for _, command := range []string{"go build ./...", "go vet ./...", "go test -race ./..."} {
				if !strings.Contains(workflow, command) {
					t.Errorf("generated CI is missing %q", command)
				}
			}
		})
	}
}

func TestGeneratedDocsAreTheSameAcrossStacks(t *testing.T) {
	firstConfig := fullSQLiteHTMXConfig()
	firstConfig.ProjectName = "Acme"
	firstConfig.Content = []Content{ContentDocs}

	secondConfig := firstConfig
	secondConfig.ModulePath = "example.com/acme-other-stack"
	secondConfig.Framework = FrameworkDatastar
	secondConfig.Database = DatabasePostgres
	secondConfig.Payment = PaymentPolar
	secondConfig.Mail = MailResend
	secondConfig.Workspaces = false
	secondConfig.OAuth = nil
	secondConfig.Storage = false

	firstDestination := filepath.Join(t.TempDir(), "first")
	secondDestination := filepath.Join(t.TempDir(), "second")
	if err := Generate(firstDestination, "3.0.0", firstConfig); err != nil {
		t.Fatalf("Generate(first) error = %v", err)
	}
	if err := Generate(secondDestination, "3.0.0", secondConfig); err != nil {
		t.Fatalf("Generate(second) error = %v", err)
	}
	firstTree := readTree(t, firstDestination)
	secondTree := readTree(t, secondDestination)
	expectedPaths := []string{
		"content/docs/introduction.md",
		"content/docs/getting-started.md",
		"content/docs/essentials/_index.md",
		"content/docs/essentials/structure.md",
		"content/docs/essentials/navigation.md",
		"content/docs/guides/_index.md",
		"content/docs/guides/workflow.md",
		"content/docs/guides/customize-the-docs.md",
		"content/docs/guides/advanced/_index.md",
		"content/docs/guides/advanced/llm-discovery.md",
		"content/docs/guides/advanced/publishing.md",
		"content/docs/faq.md",
	}
	for _, path := range expectedPaths {
		if !bytes.Equal(firstTree[path], secondTree[path]) {
			t.Errorf("generated documentation differs across stacks: %s", path)
		}
	}
	for _, tree := range []map[string][]byte{firstTree, secondTree} {
		count := 0
		var content strings.Builder
		for path, value := range tree {
			if strings.HasPrefix(path, "content/docs/") {
				count++
				content.Write(value)
			}
		}
		if count != len(expectedPaths) {
			t.Errorf("generated %d docs files, want %d", count, len(expectedPaths))
		}
		allDocs := content.String()
		for _, expected := range []string{"```bash", "```go", "```json", "/llms.txt"} {
			if !strings.Contains(allDocs, expected) {
				t.Errorf("generated docs do not contain %q", expected)
			}
		}
		for _, forbidden := range []string{"Stripe", "Polar", "Datastar", "htmx", "PostgreSQL", "SQLite", "goilerplate"} {
			if strings.Contains(allDocs, forbidden) {
				t.Errorf("generated docs contain stack-specific product term %q", forbidden)
			}
		}
	}
}

func TestStripeGenerationAlwaysEnablesManagedPayments(t *testing.T) {
	for _, workspaces := range []bool{false, true} {
		config := fullSQLiteHTMXConfig()
		config.Payment = PaymentStripe
		config.Workspaces = workspaces
		destination := filepath.Join(t.TempDir(), fmt.Sprintf("workspaces-%t", workspaces))
		if err := Generate(destination, "3.0.0", config); err != nil {
			t.Fatalf("Generate(workspaces=%t) error = %v", workspaces, err)
		}
		tree := readTree(t, destination)
		stripeProvider := string(tree["internal/service/payment/stripe.go"])
		if !strings.Contains(stripeProvider, "ManagedPayments:") || !strings.Contains(stripeProvider, "Enabled: stripe.Bool(true)") {
			t.Fatalf("Stripe provider does not enable Managed Payments for workspaces=%t", workspaces)
		}
	}
}

func TestCuratedConfigsMatchGoldenTrees(t *testing.T) {
	configs, err := filepath.Glob("testdata/configs/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 16 {
		t.Fatalf("curated configs have %d combinations, want 16", len(configs))
	}
	for _, configPath := range configs {
		configPath := configPath
		name := strings.TrimSuffix(filepath.Base(configPath), ".json")
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			var config Config
			if err := json.Unmarshal(content, &config); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains("-"+name+"-", "-"+string(config.Framework)+"-") {
				t.Fatalf("config name %q does not expose framework %q", name, config.Framework)
			}
			first := filepath.Join(t.TempDir(), "first")
			second := filepath.Join(t.TempDir(), "second")
			if err := Generate(first, "3.0.0-test", config); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if err := Generate(second, "3.0.0-test", config); err != nil {
				t.Fatalf("second Generate() error = %v", err)
			}
			template, err := CurrentTemplate("3.0.0-test")
			if err != nil {
				t.Fatal(err)
			}
			preview, err := template.PreviewFiles(config)
			if err != nil {
				t.Fatalf("PreviewFiles() error = %v", err)
			}
			generatedTree := readTree(t, first)
			assertGeneratedFormPendingStates(t, generatedTree, config.Framework)
			assertGeneratedFrameworkMarkup(t, generatedTree, config.Framework)
			generatedPaths := make([]string, 0, len(generatedTree))
			for path := range generatedTree {
				generatedPaths = append(generatedPaths, path)
			}
			sort.Strings(generatedPaths)
			if !reflect.DeepEqual(preview, generatedPaths) {
				t.Fatal("file preview differs from generated tree")
			}
			firstManifest := treeManifest(t, first)
			if firstManifest != treeManifest(t, second) {
				t.Fatal("identical inputs produced different trees")
			}
			goldenPath := filepath.Join("testdata", "golden", name+".sha256")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, []byte(firstManifest), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden manifest: %v", err)
			}
			if firstManifest != string(golden) {
				t.Fatalf("generated tree differs from %s. Review the change, then run UPDATE_GOLDEN=1 go test ./...", goldenPath)
			}
		})
	}
}

var formTagPattern = regexp.MustCompile(`(?s)<form\b.*?>`)

func assertGeneratedFormPendingStates(t *testing.T, tree map[string][]byte, framework Framework) {
	t.Helper()
	for path, content := range tree {
		if filepath.Ext(path) != ".templ" || path == "internal/ui/csrf/csrf.templ" {
			continue
		}
		for _, form := range formTagPattern.FindAll(content, -1) {
			switch framework {
			case FrameworkHTMX:
				if !bytes.Contains(form, []byte("hx-")) {
					continue
				}
				if !bytes.Contains(form, []byte("hx-disable")) {
					t.Errorf("%s contains an HTMX form without a disabled state: %s", path, form)
				}
			case FrameworkDatastar:
				if !bytes.Contains(form, []byte("data-on:")) {
					continue
				}
				if !bytes.Contains(form, []byte("data-indicator")) {
					t.Errorf("%s contains a Datastar form without a request indicator: %s", path, form)
					continue
				}
				if bytes.Contains(form, []byte(`class="hidden"`)) {
					continue
				}
				for _, attribute := range []string{
					"data-attr:inert",
					"data-attr:aria-busy",
				} {
					if !bytes.Contains(form, []byte(attribute)) {
						t.Errorf("%s contains a Datastar form without %s: %s", path, attribute, form)
					}
				}
			}
		}

		lines := bytes.Split(content, []byte("\n"))
		for index, line := range lines {
			switch framework {
			case FrameworkHTMX:
				if bytes.Contains(line, []byte(`"hx-post"`)) ||
					bytes.Contains(line, []byte(`"hx-put"`)) ||
					bytes.Contains(line, []byte(`"hx-patch"`)) ||
					bytes.Contains(line, []byte(`"hx-delete"`)) {
					end := index + 8
					if end > len(lines) {
						end = len(lines)
					}
					if !bytes.Contains(bytes.Join(lines[index:end], []byte("\n")), []byte(`"hx-disable"`)) {
						t.Errorf("%s:%d contains a direct HTMX action without hx-disable", path, index+1)
					}
				}
			case FrameworkDatastar:
				if bytes.Contains(line, []byte(`"data-on:click"`)) &&
					(bytes.Contains(line, []byte("@post(")) ||
						bytes.Contains(line, []byte("@put(")) ||
						bytes.Contains(line, []byte("@patch(")) ||
						bytes.Contains(line, []byte("@delete("))) {
					end := index + 8
					if end > len(lines) {
						end = len(lines)
					}
					action := bytes.Join(lines[index:end], []byte("\n"))
					for _, attribute := range []string{`"data-indicator`, `"data-attr:disabled"`, `"data-attr:aria-busy"`} {
						if !bytes.Contains(action, []byte(attribute)) {
							t.Errorf("%s:%d contains a direct Datastar action without %s\"", path, index+1, attribute)
						}
					}
					signal := datastarIndicatorSignal(action)
					collapsed := datastarWhitespacePattern.ReplaceAll(action, []byte(" "))
					for _, attribute := range []string{"disabled", "aria-busy"} {
						reference := fmt.Sprintf(`"data-attr:%s": "$%s"`, attribute, signal)
						if signal != "" && !bytes.Contains(collapsed, []byte(reference)) {
							t.Errorf("%s:%d sets the indicator signal $%s but data-attr:%s reads another signal", path, index+1, signal, attribute)
						}
					}
				}
			}
		}
	}
}

var datastarWhitespacePattern = regexp.MustCompile(`\s+`)
var datastarIndicatorKeyPattern = regexp.MustCompile(`"data-indicator:([a-zA-Z0-9_-]+)"`)
var datastarIndicatorValuePattern = regexp.MustCompile(`"data-indicator":\s*"([^"]+)"`)

// datastarIndicatorSignal returns the signal a data-indicator attribute in
// templ attribute syntax sets. Datastar 1.0 camel cases the key form
// (data-indicator:_remove-avatar sets $_removeAvatar) and uses the value form
// literally (data-indicator="_remove-avatar" sets a signal with a dash).
func datastarIndicatorSignal(action []byte) string {
	match := datastarIndicatorKeyPattern.FindSubmatch(action)
	if match != nil {
		parts := strings.Split(string(match[1]), "-")
		for index := 1; index < len(parts); index++ {
			if parts[index] != "" {
				parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
			}
		}
		return strings.Join(parts, "")
	}
	match = datastarIndicatorValuePattern.FindSubmatch(action)
	if match != nil {
		return string(match[1])
	}
	return ""
}

// booleanAriaInvalidPattern matches "aria-invalid": with a comparison as the
// value. aria-invalid must render "true" or "false".
var booleanAriaInvalidPattern = regexp.MustCompile(`"aria-invalid":\s*[^",}(]*(?:!=|==)`)

var frameworkAttributePattern = regexp.MustCompile(`(?:hx-[a-z-]+|data-(?:on|indicator|attr|class)(?::[a-zA-Z0-9_-]+)?)`)
var datastarCamelCaseIndicatorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`data-indicator:_[^\s=>"]*[A-Z]`),
	regexp.MustCompile(`"data-indicator":\s*"_[^"]*[A-Z]`),
}

func assertGeneratedFrameworkMarkup(t *testing.T, tree map[string][]byte, framework Framework) {
	t.Helper()
	if framework == FrameworkHeadless || framework == FrameworkSvelte {
		for path := range tree {
			if filepath.Ext(path) == ".templ" {
				t.Errorf("%s output contains templ markup %s", framework, path)
			}
		}
		return
	}
	for path, content := range tree {
		if filepath.Ext(path) != ".templ" {
			continue
		}
		for index, line := range bytes.Split(content, []byte("\n")) {
			if framework == FrameworkDatastar {
				for _, pattern := range datastarCamelCaseIndicatorPatterns {
					if pattern.Match(line) {
						t.Errorf("%s:%d uses a camelCase Datastar indicator key that HTML lowercases: %s", path, index+1, line)
					}
				}
			}
			if bytes.Contains(line, []byte("<")) && len(frameworkAttributePattern.FindAll(line, -1)) > 1 {
				t.Errorf("%s:%d contains multiple framework attributes on one line: %s", path, index+1, line)
			}
			if booleanAriaInvalidPattern.Match(line) {
				t.Errorf("%s:%d renders aria-invalid from a Go bool, which templ writes as a bare attribute that styles read as false: %s", path, index+1, line)
			}
			if bytes.Contains(line, []byte("data-class:goilerplate-loading")) {
				t.Errorf("%s:%d uses a redundant Datastar loading class: %s", path, index+1, line)
			}
			if bytes.Contains(line, []byte("hx-indicator")) && !bytes.Contains(line, []byte(`hx-indicator="#delete-project-button"`)) {
				t.Errorf("%s:%d uses a redundant HTMX indicator target: %s", path, index+1, line)
			}
		}
	}

	appLayout := tree["internal/ui/layouts/app.templ"]
	if bytes.Contains(appLayout, []byte(`/auth/logout`)) {
		for _, expected := range []string{
			`action="/auth/logout"`,
			`method="POST"`,
			`dropdown.ItemProps{Type: "submit"}`,
		} {
			if !bytes.Contains(appLayout, []byte(expected)) {
				t.Errorf("logout form is missing %s", expected)
			}
		}
		if bytes.Contains(appLayout, []byte(`"type": "submit"`)) {
			t.Error("logout passes a duplicate type attribute through templ.Attributes")
		}
	}

	if framework == FrameworkDatastar {
		baseLayout := tree["internal/ui/layouts/base.templ"]
		if !bytes.Contains(baseLayout, []byte(`data-preserve-attr="class"`)) {
			t.Error("Datastar base layout does not preserve the runtime theme class")
		}
	}
}

var nativeFrontendWord = regexp.MustCompile(`(?i)\btempl\b|tailwind|datastar|htmx`)

// htmlFrontendWord is the SvelteKit variant: its app uses Tailwind CSS, so
// only the templ stack words count.
var htmlFrontendWord = regexp.MustCompile(`(?i)\btempl\b|datastar|htmx`)

// TestHeadlessProjectsHaveNoNativeFrontend covers Headless and SvelteKit. A
// SvelteKit project keeps its app under web/; everything outside it is the
// same backend without templ, Tailwind, htmx or Datastar.
func TestHeadlessProjectsHaveNoNativeFrontend(t *testing.T) {
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"assets/", "content/", "internal/ui/", "internal/handler/", "internal/content/", "internal/markdown/"}
	generated := make(map[Framework]int)
	for _, config := range templateTestConfigs() {
		if config.HasHTMLFrontend() {
			continue
		}
		files, err := template.plan(config)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if config.IsSvelte() && strings.HasPrefix(file.Destination, "web/") {
				continue
			}
			for _, prefix := range forbidden {
				if strings.HasPrefix(file.Destination, prefix) {
					t.Errorf("%s: %s selects %s", config.ModulePath, config.Framework, file.Source)
				}
			}
			if file.Destination == "components.json" || filepath.Ext(file.Destination) == ".templ" {
				t.Errorf("%s: %s selects %s", config.ModulePath, config.Framework, file.Source)
			}
		}
		minimal := !config.Workspaces && !config.Storage && len(config.OAuth) == 0 && config.Database == DatabaseSQLite && config.Payment == PaymentStripe && config.Mail == MailSMTP
		if minimal || (config.Workspaces && config.Storage && len(config.OAuth) == 2) {
			generated[config.Framework]++
			destination := filepath.Join(t.TempDir(), "project")
			err := template.Generate(destination, config)
			if err != nil {
				t.Fatal(err)
			}
			for path, content := range readTree(t, destination) {
				if path == "go.sum" || (config.IsSvelte() && strings.HasPrefix(path, "web/")) {
					continue
				}
				words := nativeFrontendWord
				if config.IsSvelte() {
					words = htmlFrontendWord
				}
				match := words.Find(content)
				if match != nil {
					t.Errorf("%s: %s %s mentions %q", config.ModulePath, config.Framework, path, match)
				}
			}
		}
	}
	if generated[FrameworkHeadless] == 0 || generated[FrameworkSvelte] == 0 {
		t.Fatalf("generated projects = %v, want Headless and SvelteKit", generated)
	}
}

func TestTemplateConditionsAreKnown(t *testing.T) {
	err := filepath.WalkDir("_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "_template" {
			return nil
		}
		relative, err := filepath.Rel("_template", path)
		if err != nil {
			return err
		}
		if _, _, err := splitConditions(filepath.ToSlash(relative)); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDestinationVariantsAreDistinct(t *testing.T) {
	type sourceFile struct {
		path    string
		content []byte
	}
	byDestination := make(map[string][]sourceFile)
	err := filepath.WalkDir("_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("_template", path)
		if err != nil {
			return err
		}
		destination, _, err := splitConditions(filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") {
			content = bytes.ReplaceAll(content, []byte("{{.ModulePath}}"), []byte("example.com/project"))
			content = bytes.ReplaceAll(content, []byte("{{.ProjectNameGo}}"), []byte(`"Project"`))
			content = bytes.ReplaceAll(content, []byte("{{.DatabaseTimestamp}}"), []byte("TIMESTAMP"))
			content, err = format.Source(content)
			if err != nil {
				return fmt.Errorf("format %s: %w", path, err)
			}
		}
		byDestination[destination] = append(byDestination[destination], sourceFile{path: path, content: content})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for destination, sources := range byDestination {
		for left := range sources {
			for right := left + 1; right < len(sources); right++ {
				if bytes.Equal(sources[left].content, sources[right].content) {
					t.Errorf("%s has identical sources %s and %s", destination, sources[left].path, sources[right].path)
				}
			}
		}
	}
}

func TestConditionsAreMinimal(t *testing.T) {
	configs := templateTestConfigs()
	axisOrder := make(map[string]int, len(wordTable))
	for index, row := range wordTable {
		axisOrder[row.Axis] = index
	}
	used := make(map[string]bool)
	directoryWordChanges := make(map[string]bool)

	err := filepath.WalkDir("_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("_template", path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		_, conditions, err := splitConditions(relative)
		if err != nil {
			return err
		}
		for _, occurrences := range conditionWords(relative) {
			previousOrder := -1
			previousWord := -1
			for _, word := range occurrences {
				used[word] = true
				order := axisOrder[words[word]]
				wordOrder := slices.Index(wordTable[order].Words, word)
				if order < previousOrder || (order == previousOrder && wordOrder < previousWord) {
					t.Errorf("%s has condition words outside table order", relative)
					break
				}
				previousOrder = order
				previousWord = wordOrder
			}
		}

		// A file name word must change the selection of its file. A directory
		// word must change the selection of at least one file below it.
		selected := matchingConfigs(configs, conditions, "")
		elements := strings.Split(relative, "/")
		// Removing the words of web.svelte/ would make a literal file name
		// like app.d.ts parse as words, so probe with a plain file name.
		probe := relative
		if hasLiteralFileName(relative) {
			probe = strings.Join(elements[:len(elements)-1], "/") + "/literal"
		}
		for elementIndex, elementWords := range conditionWords(relative) {
			axes := make(map[string]bool)
			for _, word := range elementWords {
				axes[words[word]] = true
			}
			for axis := range axes {
				_, withoutConditions, err := splitConditions(withoutAxisWords(probe, elementIndex, axis))
				if err != nil {
					return err
				}
				changes := !reflect.DeepEqual(selected, matchingConfigs(configs, withoutConditions, ""))
				if elementIndex == len(elements)-1 {
					if !changes {
						t.Errorf("%s condition %q does not change its selected configs", relative, axis)
					}
					continue
				}
				directory := strings.Join(elements[:elementIndex+1], "/") + " " + axis
				directoryWordChanges[directory] = directoryWordChanges[directory] || changes
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for directory, changes := range directoryWordChanges {
		if !changes {
			t.Errorf("directory condition %s does not change the selection of any file below it", directory)
		}
	}

	for word := range words {
		if used[word] {
			continue
		}
		opposite := "no_" + word
		if strings.HasPrefix(word, "no_") {
			opposite = strings.TrimPrefix(word, "no_")
		}
		if !used[opposite] {
			t.Errorf("condition word %q and its opposite are unused", word)
		}
	}
}

func matchingConfigs(configs []Config, conditions map[string][]string, omittedAxis string) []int {
	var selected []int
	for index, config := range configs {
		configFacts := facts(config)
		matches := true
		for axis, allowed := range conditions {
			if axis != omittedAxis && !slices.Contains(allowed, configFacts[axis]) {
				matches = false
				break
			}
		}
		if matches {
			selected = append(selected, index)
		}
	}
	return selected
}

// withoutAxisWords removes the words of one axis from one path element.
func withoutAxisWords(path string, elementIndex int, axis string) string {
	elements := strings.Split(path, "/")
	if hasLiteralFileName(path) && elementIndex == len(elements)-1 {
		return path
	}
	segments := strings.Split(elements[elementIndex], ".")
	kept := make([]string, 0, len(segments))
	for index, segment := range segments {
		if index > 0 && words[segment] == axis {
			continue
		}
		kept = append(kept, segment)
	}
	elements[elementIndex] = strings.Join(kept, ".")
	return strings.Join(elements, "/")
}

// conditionWords returns the condition words of every path element.
func conditionWords(path string) [][]string {
	var result [][]string
	elements := strings.Split(filepath.ToSlash(path), "/")
	literalFileName := hasLiteralFileName(filepath.ToSlash(path))
	for elementIndex, element := range elements {
		if literalFileName && elementIndex == len(elements)-1 {
			result = append(result, nil)
			continue
		}
		segments := strings.Split(element, ".")
		baseSegments := 1
		if len(segments) > 1 && segments[0] == "" {
			baseSegments = 2
		}
		extensionSegments := 0
		if elementIndex == len(elements)-1 && len(segments) > baseSegments {
			if _, conditionWithoutExtension := words[segments[len(segments)-1]]; !conditionWithoutExtension {
				extensionSegments = 1
				if segments[len(segments)-1] == "tmpl" && len(segments) > baseSegments+1 {
					extensionSegments = 2
				}
			}
		}
		end := len(segments) - extensionSegments
		var elementWords []string
		for index := baseSegments; index < end; index++ {
			_, ok := words[segments[index]]
			if ok {
				elementWords = append(elementWords, segments[index])
			}
		}
		result = append(result, elementWords)
	}
	return result
}

func TestNoDestinationCollisions(t *testing.T) {
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range templateTestConfigs() {
		normalized, err := config.normalized()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := template.plan(normalized); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEveryConfigGeneratesOneInitialMigration(t *testing.T) {
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range templateTestConfigs() {
		files, err := template.plan(config)
		if err != nil {
			t.Fatal(err)
		}
		var migrations []string
		for _, file := range files {
			if strings.HasPrefix(file.Destination, "internal/db/migrations/") && strings.HasSuffix(file.Destination, ".sql") {
				migrations = append(migrations, file.Destination)
			}
		}
		if !reflect.DeepEqual(migrations, []string{"internal/db/migrations/001_initial_schema.sql"}) {
			t.Errorf("%s generated migrations %v", config.ModulePath, migrations)
		}
	}
}

func TestEveryTemplateFileIsSelected(t *testing.T) {
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	selected := make(map[string]struct{})
	for _, config := range templateTestConfigs() {
		normalized, err := config.normalized()
		if err != nil {
			t.Fatal(err)
		}
		files, err := template.plan(normalized)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			selected[file.Source] = struct{}{}
		}
	}
	if err := fs.WalkDir(template.files, "_template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if _, ok := selected[path]; !ok {
			t.Errorf("template source %s is never selected", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func templateTestConfigs() []Config {
	var configs []Config
	for _, config := range configuratorSelections() {
		for _, framework := range configuratorFrontends(config) {
			frameworkConfig := config
			frameworkConfig.Framework = framework
			normalized, err := frameworkConfig.normalized()
			if err != nil {
				panic(err)
			}
			configs = append(configs, normalized)
		}
	}
	return configs
}

func treeManifest(t *testing.T, root string) string {
	t.Helper()
	tree := readTree(t, root)
	paths := make([]string, 0, len(tree))
	for path := range tree {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var manifest strings.Builder
	for _, path := range paths {
		digest := sha256.Sum256(tree[path])
		fmt.Fprintf(&manifest, "%x  %s\n", digest, path)
	}
	return manifest.String()
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatalf("walk tree: %v", err)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		result[relative] = bytes.Clone(content)
	}
	return result
}

// TestMCPDocsCarryTheTestedToolExample keeps the add-a-tool example in the
// shipped README.md and AGENTS.md identical to internal/mcp/example_test.go,
// where it runs against a real server.
func TestMCPDocsCarryTheTestedToolExample(t *testing.T) {
	minimal := minimalConfig()
	minimal.API = true
	minimal.MCP = true
	headless := fullSQLiteHTMXConfig()
	headless.Framework = FrameworkHeadless
	headless.Content = nil
	headless.API = true
	headless.MCP = true
	for name, config := range map[string]Config{"minimal": minimal, "headless workspaces": headless} {
		destination := filepath.Join(t.TempDir(), "project")
		err := Generate(destination, "3.1.0", config)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		tree := readTree(t, destination)
		source := string(tree["internal/mcp/example_test.go"])
		start := strings.Index(source, "type nameInput struct {")
		end := strings.Index(source, "\t}))\n}\n")
		if start < 0 || end < start {
			t.Fatalf("%s: example tool not found in example_test.go", name)
		}
		example := source[start : end+len("\t}))\n}\n")]
		for _, path := range []string{"README.md", "AGENTS.md"} {
			if !strings.Contains(string(tree[path]), "## MCP server") || !strings.Contains(string(tree[path]), example) {
				t.Errorf("%s: %s does not carry the tested add-a-tool example", name, path)
			}
		}
	}

	minimal.MCP = false
	destination := filepath.Join(t.TempDir(), "project")
	err := Generate(destination, "3.1.0", minimal)
	if err != nil {
		t.Fatal(err)
	}
	tree := readTree(t, destination)
	for _, path := range []string{"README.md", "AGENTS.md", "content/llms.txt"} {
		if strings.Contains(string(tree[path]), "MCP") {
			t.Errorf("%s mentions MCP without the MCP server", path)
		}
	}
}
