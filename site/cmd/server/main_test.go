package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteServesPagesRedirectsAndTheRetiredAPI(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"index.html":             "home",
		"docs.html":              "docs",
		"docs/installation.html": "install",
		"docs/installation.md":   "# Installation",
		"404.html":               "missing",
	} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler := newSite(dir)
	for _, test := range []struct {
		method, path string
		status       int
		body         string
		location     string
	}{
		{method: "GET", path: "/", status: 200, body: "home"},
		{method: "GET", path: "/docs", status: 200, body: "docs"},
		{method: "GET", path: "/docs/installation", status: 200, body: "install"},
		{method: "GET", path: "/docs/installation.md", status: 200, body: "# Installation"},
		{method: "GET", path: "/docs/", status: 301, location: "/docs"},
		{method: "GET", path: "/pricing", status: 301, location: "/"},
		{method: "GET", path: "/blog/some-post", status: 301, location: "/"},
		{method: "GET", path: "/ops", status: 301, location: "https://ops.goilerplate.com"},
		{method: "GET", path: "/docs/guides/cli-accounts", status: 301, location: "/docs/installation"},
		{method: "GET", path: "/nothing", status: 404, body: "missing"},
		{method: "GET", path: "/../etc/passwd", status: 404, body: "missing"},
		{method: "POST", path: "/v1/generate", status: 410, body: "go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest"},
		{method: "GET", path: "/healthz", status: 204},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s %s = %d, want %d", test.method, test.path, response.Code, test.status)
		}
		if test.body != "" && !strings.Contains(response.Body.String(), test.body) {
			t.Fatalf("%s %s body = %q, want %q", test.method, test.path, response.Body.String(), test.body)
		}
		if test.location != "" && response.Header().Get("Location") != test.location {
			t.Fatalf("%s %s location = %q, want %q", test.method, test.path, response.Header().Get("Location"), test.location)
		}
	}
}
