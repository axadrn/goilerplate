package generator

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitConditions(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantPath       string
		wantConditions map[string][]string
		wantError      string
	}{
		{name: "plain path", path: "internal/handler/account.go", wantPath: "internal/handler/account.go", wantConditions: map[string][]string{}},
		{name: "one word", path: "internal/handler/account.htmx.go", wantPath: "internal/handler/account.go", wantConditions: map[string][]string{"framework": {"htmx"}}},
		{name: "two words", path: "internal/ui/pages/dashboard.datastar.workspaces.templ", wantPath: "internal/ui/pages/dashboard.templ", wantConditions: map[string][]string{"framework": {"datastar"}, "workspaces": {"workspaces"}}},
		{name: "directory word", path: "internal/access.workspaces/catalog.go", wantPath: "internal/access/catalog.go", wantConditions: map[string][]string{"workspaces": {"workspaces"}}},
		{name: "tmpl extension", path: "internal/config/config.storage.go.tmpl", wantPath: "internal/config/config.go.tmpl", wantConditions: map[string][]string{"storage": {"storage"}}},
		{name: "word before tmpl", path: "Makefile.blog.tmpl", wantPath: "Makefile.tmpl", wantConditions: map[string][]string{"blog": {"blog"}}},
		{name: "hidden word before tmpl", path: ".env.htmx.tmpl", wantPath: ".env.tmpl", wantConditions: map[string][]string{"framework": {"htmx"}}},
		{name: "plain double extension", path: "cmd/server/main.go.tmpl", wantPath: "cmd/server/main.go.tmpl", wantConditions: map[string][]string{}},
		{name: "version segments", path: "assets/js/htmx-4.0.0.e484d917.min.htmx.js", wantPath: "assets/js/htmx-4.0.0.e484d917.min.js", wantConditions: map[string][]string{"framework": {"htmx"}}},
		{name: "hidden directory", path: ".github/workflows/ci.yml", wantPath: ".github/workflows/ci.yml", wantConditions: map[string][]string{}},
		{name: "hidden double extension", path: ".env.example.tmpl", wantPath: ".env.example.tmpl", wantConditions: map[string][]string{}},
		{name: "headless word", path: "internal/routes/routes.headless.go.tmpl", wantPath: "internal/routes/routes.go.tmpl", wantConditions: map[string][]string{"framework": {"headless"}}},
		{name: "api directory word", path: "internal/api.api/auth.go", wantPath: "internal/api/auth.go", wantConditions: map[string][]string{"api": {"api"}}},
		{name: "no_api word", path: "internal/middleware/cors.no_api.go", wantPath: "internal/middleware/cors.go", wantConditions: map[string][]string{"api": {"no_api"}}},
		{name: "framework and api words", path: "internal/handler/account.htmx.api.go", wantPath: "internal/handler/account.go", wantConditions: map[string][]string{"framework": {"htmx"}, "api": {"api"}}},
		{name: "api word before tmpl", path: "README.api.md.tmpl", wantPath: "README.md.tmpl", wantConditions: map[string][]string{"api": {"api"}}},
		{name: "alternative words", path: "assets.htmx.datastar/css/app.css", wantPath: "assets/css/app.css", wantConditions: map[string][]string{"framework": {"htmx", "datastar"}}},
		{name: "alternative file words", path: "Dockerfile.htmx.datastar", wantPath: "Dockerfile", wantConditions: map[string][]string{"framework": {"htmx", "datastar"}}},
		{name: "narrowed alternatives", path: "internal/ui/layouts.htmx.datastar/base.htmx.workspaces.templ", wantPath: "internal/ui/layouts/base.templ", wantConditions: map[string][]string{"framework": {"htmx"}, "workspaces": {"workspaces"}}},
		{name: "vendored directory", path: "internal/ui/components/button/button.templ", wantPath: "internal/ui/components/button/button.templ", wantConditions: map[string][]string{"framework": {"htmx", "datastar"}}},
		{name: "shared headless and svelte", path: "internal/api/root.headless.svelte.go", wantPath: "internal/api/root.go", wantConditions: map[string][]string{"framework": {"headless", "svelte"}}},
		{name: "literal svelte config", path: "web.svelte/svelte.config.js", wantPath: "web/svelte.config.js", wantConditions: map[string][]string{"framework": {"svelte"}}},
		{name: "literal svelte module", path: "web.svelte/src/lib/x.svelte.ts", wantPath: "web/src/lib/x.svelte.ts", wantConditions: map[string][]string{"framework": {"svelte"}}},
		{name: "literal type declarations", path: "web.svelte/src/app.d.ts", wantPath: "web/src/app.d.ts", wantConditions: map[string][]string{"framework": {"svelte"}}},
		{name: "literal file in a worded directory", path: "web.svelte/src/routes/app.workspaces/+page.svelte", wantPath: "web/src/routes/app/+page.svelte", wantConditions: map[string][]string{"framework": {"svelte"}, "workspaces": {"workspaces"}}},
		{name: "literal file keeps tmpl", path: "web.svelte/src/lib/config.ts.tmpl", wantPath: "web/src/lib/config.ts.tmpl", wantConditions: map[string][]string{"framework": {"svelte"}}},
		{name: "literal file name ignores words", path: "web.svelte/src/lib/plan.htmx.ts", wantPath: "web/src/lib/plan.htmx.ts", wantConditions: map[string][]string{"framework": {"svelte"}}},
		{name: "worded directory in web", path: "web.svelte/src/routes/billing.workspaces/+page.svelte", wantPath: "web/src/routes/billing/+page.svelte", wantConditions: map[string][]string{"framework": {"svelte"}, "workspaces": {"workspaces"}}},
		{name: "repeated word", path: "Dockerfile.htmx.htmx", wantError: "appears twice"},
		{name: "contradicting api axis", path: "internal/api.api/auth.no_api.go", wantError: "has no word that matches"},
		{name: "unknown word", path: "navbar.blogg.templ", wantError: "unknown condition word"},
		{name: "contradicting axis", path: "internal.htmx/account.datastar.go", wantError: "has no word that matches"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotPath, gotConditions, err := splitConditions(test.path)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("splitConditions() error = %v, want containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != test.wantPath {
				t.Errorf("destination = %q, want %q", gotPath, test.wantPath)
			}
			if !reflect.DeepEqual(gotConditions, test.wantConditions) {
				t.Errorf("conditions = %#v, want %#v", gotConditions, test.wantConditions)
			}
		})
	}
}

func TestConditionWordsBelongToOneAxis(t *testing.T) {
	seen := make(map[string]string)
	for _, row := range wordTable {
		for _, word := range row.Words {
			if previous, exists := seen[word]; exists {
				t.Errorf("word %q belongs to %s and %s", word, previous, row.Axis)
			}
			seen[word] = row.Axis
			if words[word] != row.Axis {
				t.Errorf("words[%q] = %q, want %q", word, words[word], row.Axis)
			}
		}
	}
}
