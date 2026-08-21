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
		Answers: api.GenerationAnswers{
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
		Answers:         api.GenerationAnswers{ModulePath: "example.com/wanted", Edition: "free", Database: "sqlite", Mail: "resend"},
	})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/wrong\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inspector := fakeInspector(map[string]string{"go": "go1.24.0", "git": "git version 2.37.0"})

	report := inspector.Inspect(context.Background(), root)
	if report.Errors != 5 {
		t.Fatalf("errors = %d, checks = %#v", report.Errors, report.Checks)
	}
	for _, name := range []string{"go.mod", "go", "git", "task", "tailwindcss"} {
		if !hasCheck(report, name, LevelError) {
			t.Fatalf("missing failed check %q in %#v", name, report.Checks)
		}
	}
}

func TestInspectValidatesSelectedEnvironment(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, api.ProjectLock{
		SchemaVersion:   api.LockSchemaVersion,
		TemplateVersion: "v3.0.0",
		Answers: api.GenerationAnswers{
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
		"APP_NAME=Acme", "APP_ENV=development", "APP_URL=http://localhost:8090", "DB_CONNECTION=./data/app.db",
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

	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("APP_NAME=Acme\n"), 0o600); err != nil {
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
		"older": {"2.37.9", "2.38.0", -1},
		"equal": {"1.25", "1.25.0", 0},
		"newer": {"1.25.7", "1.25.0", 1},
	} {
		t.Run(name, func(t *testing.T) {
			if got := compareVersions(test.left, test.right); got != test.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+lock.Answers.ModulePath+"\n\ngo 1.25.0\n"), 0o644); err != nil {
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
