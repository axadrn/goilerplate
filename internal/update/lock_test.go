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
  "answers": {
    "project_name": "Acme",
    "module_path": "example.com/acme",
    "edition": "paid",
    "framework": "htmx",
    "database": "sqlite",
    "payment": "stripe",
    "mail": "smtp",
    "teams": false,
    "oauth": [],
    "storage": false,
    "content": [],
    "example": false
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
	if lock.TemplateVersion != "v3.0.0" || lock.Answers.ModulePath != "example.com/acme" {
		t.Fatalf("lock = %#v", lock)
	}
}

func TestReadLockRejectsMissingUnknownAndUnsupportedFiles(t *testing.T) {
	if _, err := ReadLock(t.TempDir()); err == nil {
		t.Fatal("missing lock was accepted")
	}
	for name, content := range map[string]string{
		"unknown":       `{"schema_version":1,"template_version":"v3.0.0","answers":{},"extra":true}`,
		"unsupported":   `{"schema_version":2,"template_version":"v3.0.0","answers":{}}`,
		"empty version": `{"schema_version":1,"template_version":" ","answers":{}}`,
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
