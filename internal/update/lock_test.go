package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLock(t *testing.T) {
	root := t.TempDir()
	content := `{
  "schema_version": 1,
  "template_version": "v3.0.0",
  "config": {
    "project_name": "Acme",
    "module_path": "example.com/acme",
    "edition": "paid",
    "framework": "htmx",
    "database": "sqlite",
    "payment": "stripe",
    "mail": "smtp",
    "workspaces": false,
    "oauth": [],
    "storage": false,
    "content": []
  }
}
`
	if err := os.WriteFile(filepath.Join(root, "goilerplate.lock"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := ReadLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if lock.TemplateVersion != "v3.0.0" || lock.Config.ModulePath != "example.com/acme" {
		t.Fatalf("lock = %#v", lock)
	}
}

func TestReadLockDecodesTheJSONAPI(t *testing.T) {
	for name, test := range map[string]struct {
		config string
		want   bool
	}{
		"older lock without api": {config: `{"framework":"htmx"}`, want: false},
		"api off":                {config: `{"framework":"datastar","api":false}`, want: false},
		"api on":                 {config: `{"framework":"htmx","api":true}`, want: true},
		"headless":               {config: `{"framework":"headless","api":true}`, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			content := `{"schema_version":1,"template_version":"v3.0.0","config":` + test.config + `}`
			err := os.WriteFile(filepath.Join(root, "goilerplate.lock"), []byte(content), 0o644)
			if err != nil {
				t.Fatal(err)
			}
			lock, err := ReadLock(root)
			if err != nil {
				t.Fatal(err)
			}
			if lock.Config.API != test.want {
				t.Fatalf("api = %v, want %v", lock.Config.API, test.want)
			}
		})
	}
}

func TestReadLockRejectsMissingUnknownAndUnsupportedFiles(t *testing.T) {
	if _, err := ReadLock(t.TempDir()); err == nil {
		t.Fatal("missing lock was accepted")
	}
	for name, content := range map[string]string{
		"unknown":       `{"schema_version":1,"template_version":"v3.0.0","config":{},"extra":true}`,
		"unsupported":   `{"schema_version":2,"template_version":"v3.0.0","config":{}}`,
		"empty version": `{"schema_version":1,"template_version":" ","config":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "goilerplate.lock"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadLock(root); err == nil || strings.TrimSpace(err.Error()) == "" {
				t.Fatalf("invalid lock error = %v", err)
			}
		})
	}
}
