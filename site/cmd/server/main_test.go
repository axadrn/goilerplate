package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteRendersPagesRedirectsAndTheRetiredAPI(t *testing.T) {
	// The CSS is built by Tailwind, which the Go tests do not run.
	assets := t.TempDir()
	for _, name := range []string{"css/output.css", "css/globals.css"} {
		file := filepath.Join(assets, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("body{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler, err := newSite("../../../content", "../../../CHANGELOG.md", assets, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method, path string
		status       int
		body         string
		location     string
	}{
		{method: "GET", path: "/", status: 200, body: "goilerplate"},
		{method: "GET", path: "/new", status: 200, body: "configurator-install"},
		{method: "GET", path: "/changelog", status: 200, body: "v4.0.0"},
		{method: "GET", path: "/docs", status: 200, body: "<html"},
		{method: "GET", path: "/docs/installation", status: 200, body: "<html"},
		{method: "GET", path: "/docs/installation.md", status: 200, body: "#"},
		{method: "GET", path: "/legal/privacy", status: 200, body: "<html"},
		{method: "GET", path: "/sitemap.xml", status: 200, body: "/docs/installation</loc>"},
		{method: "GET", path: "/llms.txt", status: 200, body: "/docs/installation.md"},
		{method: "GET", path: "/robots.txt", status: 200, body: "Sitemap:"},
		{method: "GET", path: "/assets/css/output.css", status: 200},
		{method: "GET", path: "/assets/css/globals.css", status: 404},
		{method: "GET", path: "/docs/", status: 301, location: "/docs"},
		{method: "GET", path: "/pricing", status: 301, location: "/"},
		{method: "GET", path: "/blog/some-post", status: 301, location: "/"},
		{method: "GET", path: "/ops", status: 301, location: "https://ops.goilerplate.com"},
		{method: "GET", path: "/docs/guides/cli-accounts", status: 301, location: "/docs/installation"},
		{method: "GET", path: "/nothing", status: 404, body: "<html"},
		{method: "GET", path: "/../etc/passwd", status: 404},
		{method: "GET", path: "/assets/../../go.mod", status: 404},
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
			t.Fatalf("%s %s body does not contain %q", test.method, test.path, test.body)
		}
		if test.location != "" && response.Header().Get("Location") != test.location {
			t.Fatalf("%s %s location = %q, want %q", test.method, test.path, response.Header().Get("Location"), test.location)
		}
	}
}
