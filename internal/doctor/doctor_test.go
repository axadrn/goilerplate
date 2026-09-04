package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axadrn/goilerplate/v3/api"
)

func TestInspectChecksProjectAndSelectedTools(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Config: api.GenerationAnswers{
			ModulePath: "example.com/acme",
			Edition:    "paid",
			Database:   "postgres",
			Mail:       "resend",
		},
	})
	nested := filepath.Join(root, "internal", "service")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	inspector := fakeInspector(map[string]string{
		"go": "go1.25.7", "git": "git version 2.50.1", "task": "", "tailwindcss": "", "docker": "",
	})

	report := inspector.Inspect(context.Background(), nested)
	if report.Errors != 0 {
		t.Fatalf("report = %#v", report)
	}
	if !hasCheck(report, "project", LevelOK) || !hasCheck(report, "docker", LevelOK) || !hasCheck(report, ".env", LevelWarning) {
		t.Fatalf("checks = %#v", report.Checks)
	}
}

func TestInspectReportsModuleVersionsAndMissingTools(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Config:          api.GenerationAnswers{ModulePath: "example.com/wanted", Edition: "free", Database: "sqlite", Mail: "resend"},
	})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/wrong\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inspector := fakeInspector(map[string]string{"go": "go1.24.0", "git": "git version 2.37.0"})

	report := inspector.Inspect(context.Background(), root)
	if report.Errors != 3 {
		t.Fatalf("errors = %d, checks = %#v", report.Errors, report.Checks)
	}
	for _, name := range []string{"go.mod", "go", "git"} {
		if !hasCheck(report, name, LevelError) {
			t.Fatalf("missing failed check %q in %#v", name, report.Checks)
		}
	}
	for _, name := range []string{"task", "tailwindcss"} {
		if !hasCheck(report, name, LevelWarning) {
			t.Fatalf("missing optional warning %q in %#v", name, report.Checks)
		}
	}
}

func TestInspectAllowsCommentsAndNoGoVersionGate(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Config:          api.GenerationAnswers{ModulePath: "example.com/acme", Edition: "free", Database: "sqlite", Mail: "resend"},
	})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/acme // generated project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := fakeInspector(map[string]string{"git": "git version 2.50.1"}).Inspect(context.Background(), root)
	if report.Errors != 0 || hasCheckName(report, "go") {
		t.Fatalf("report = %#v", report)
	}
}

func TestInspectValidatesSelectedEnvironment(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Config: api.GenerationAnswers{
			ModulePath: "example.com/acme",
			Edition:    "paid",
			Database:   "sqlite",
			Mail:       "resend",
			Payment:    "stripe",
			OAuth:      []string{"github"},
			Storage:    true,
		},
	})
	environment := strings.Join([]string{
		"APP_ENV=development", "APP_URL=http://localhost:8090", "DB_CONNECTION=./data/app.db",
		"EMAIL_FROM=noreply@example.com", "SESSION_EXPIRY=168h", "RESEND_API_KEY=re_test", "STRIPE_SECRET_KEY=sk_test",
		"STRIPE_WEBHOOK_SECRET=whsec_test", "STRIPE_PRICE_ID_PRO_MONTHLY=price_1", "STRIPE_PRICE_ID_PRO_YEARLY=price_2",
		"STRIPE_PRICE_ID_ENTERPRISE_MONTHLY=price_3", "STRIPE_PRICE_ID_ENTERPRISE_YEARLY=price_4", "GITHUB_CLIENT_ID=id",
		"GITHUB_CLIENT_SECRET=secret", "S3_REGION=auto", "S3_BUCKET=bucket", "S3_ACCESS_KEY=access", "S3_SECRET_KEY=secret",
		"S3_ENDPOINT=https://example.com",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(environment), 0o600); err != nil {
		t.Fatal(err)
	}
	inspector := fakeInspector(map[string]string{"go": "go1.25.7", "git": "git version 2.50.1", "task": "", "tailwindcss": ""})
	if report := inspector.Inspect(context.Background(), root); report.Errors != 0 || !hasCheck(report, ".env", LevelOK) {
		t.Fatalf("report = %#v", report)
	}

	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("APP_ENV=development\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := inspector.Inspect(context.Background(), root)
	if !hasCheck(report, ".env", LevelError) {
		t.Fatalf("report = %#v", report)
	}
}

func TestInspectRejectsDirectoryWithoutLock(t *testing.T) {
	report := fakeInspector(nil).Inspect(context.Background(), t.TempDir())
	if report.Errors != 1 || len(report.Checks) != 1 || report.Checks[0].Name != "project" {
		t.Fatalf("report = %#v", report)
	}
}

func TestCompareVersions(t *testing.T) {
	for name, test := range map[string]struct {
		left, right string
		want        int
	}{
		"older":             {"2.37.9", "2.38.0", -1},
		"equal":             {"1.25", "1.25.0", 0},
		"newer":             {"1.25.7", "1.25.0", 1},
		"four components":   {"2.38.0.1", "2.38.0", 0},
		"windows suffix":    {"2.38.1.windows.1", "2.38.1", 0},
		"release candidate": {"1.26rc1", "1.26.0", 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got := compareVersions(test.left, test.right); got != test.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestFirstVersionReadsToolOutputVariants(t *testing.T) {
	for _, value := range []string{"go1.25.7", "go version go1.26rc1 windows/amd64", "git version 2.50.1.windows.1", "git version 2.50.1.2"} {
		if _, ok := firstVersion(value); !ok {
			t.Fatalf("firstVersion(%q) did not find a version", value)
		}
	}
}

func TestReadEnvironmentRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(strings.Repeat("A", maxEnvironmentSize+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEnvironment(path); err == nil || !strings.Contains(err.Error(), "larger") {
		t.Fatalf("readEnvironment() error = %v", err)
	}
}

func fakeInspector(outputs map[string]string) Inspector {
	return Inspector{
		LookPath: func(name string) (string, error) {
			if _, ok := outputs[name]; !ok {
				return "", errors.New("not found")
			}
			return "/bin/" + name, nil
		},
		Output: func(_ context.Context, name string, _ ...string) (string, error) {
			return outputs[name], nil
		},
	}
}

func writeProject(t *testing.T, root string, lock api.ProjectLock) {
	t.Helper()
	encoded, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "goilerplate.lock"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+lock.Config.ModulePath+"\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasCheck(report Report, name string, level Level) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Level == level {
			return true
		}
	}
	return false
}

func hasCheckName(report Report, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name {
			return true
		}
	}
	return false
}
