package update

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTemplateReadsTheTemplateFromTheModuleZip(t *testing.T) {
	prefix := "github.com/axadrn/goilerplate/v4@v4.1.0/"
	module := moduleZip(t, map[string]string{
		prefix + "go.mod":                              "module github.com/axadrn/goilerplate/v4\n",
		prefix + "generator/generate.go":               "package generator\n",
		prefix + "generator/_template/go.mod.tmpl":     "module {{.ModulePath}}\n",
		prefix + "generator/_template/internal/a.go":   "package internal\n",
		"github.com/axadrn/goilerplate/v4@v4.0.0/x.go": "package other\n",
	})
	var requested string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		_, _ = w.Write(module)
	}))
	defer server.Close()

	files, err := FetchTemplate(context.Background(), server.Client(), server.URL+"/", "v4.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if requested != "/github.com/axadrn/goilerplate/v4/@v/v4.1.0.zip" {
		t.Fatalf("requested %s", requested)
	}
	var names []string
	err = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			names = append(names, name)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "_template/go.mod.tmpl" || names[1] != "_template/internal/a.go" {
		t.Fatalf("template files = %v", names)
	}
	if content, _ := fs.ReadFile(files, "_template/go.mod.tmpl"); string(content) != "module {{.ModulePath}}\n" {
		t.Fatalf("go.mod.tmpl = %q", content)
	}
}

func TestFetchTemplateReportsAMissingVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := FetchTemplate(context.Background(), server.Client(), server.URL, "v4.9.9")
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func TestFetchTemplateRejectsAModuleWithoutTemplate(t *testing.T) {
	module := moduleZip(t, map[string]string{"github.com/axadrn/goilerplate/v4@v4.1.0/go.mod": "module x\n"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(module)
	}))
	defer server.Close()

	if _, err := FetchTemplate(context.Background(), server.Client(), server.URL, "v4.1.0"); err == nil {
		t.Fatal("accepted a module without a template")
	}
}

func TestModulePathFollowsTheMajorVersion(t *testing.T) {
	for version, want := range map[string]string{
		"v4.1.0":        "github.com/axadrn/goilerplate/v4",
		"v5.0.0-beta.1": "github.com/axadrn/goilerplate/v5",
		"v1.9.0":        "github.com/axadrn/goilerplate",
	} {
		if got := ModulePath(version); got != want {
			t.Fatalf("ModulePath(%s) = %s, want %s", version, got, want)
		}
	}
}

func moduleZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
