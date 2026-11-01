package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
)

func TestDocsReturnsFullPagesAndHTMXFragments(t *testing.T) {
	handler := newDocsTestHandler(t)

	full := serveDocs(t, handler, false, "/docs")
	if full.Code != http.StatusOK || !strings.Contains(strings.ToLower(full.Body.String()), "<!doctype html>") {
		t.Fatalf("full response status = %d, body does not contain a document", full.Code)
	}
	if vary := full.Header().Values("Vary"); !containsHeaderValue(vary, "HX-Request") {
		t.Fatalf("full response Vary = %q", vary)
	}
	body := full.Body.String()
	for _, want := range []string{
		`hx-sync="#docs-content-wrapper:replace"`,
		`hx-status:4xx="swap:none push:false"`,
		`hx-status:5xx="swap:none push:false"`,
		`goilerplate:docs:navigated`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("full response does not contain %q", want)
		}
	}

	fragment := serveDocs(t, handler, true, "/docs")
	if fragment.Code != http.StatusOK || strings.Contains(strings.ToLower(fragment.Body.String()), "<!doctype html>") {
		t.Fatalf("HTMX response status = %d, body is not a fragment", fragment.Code)
	}
	if !strings.Contains(fragment.Body.String(), `id="docs-content"`) || !strings.Contains(fragment.Body.String(), "hx-swap-oob") {
		t.Fatal("HTMX response is missing docs content or out-of-band updates")
	}
	if vary := fragment.Header().Values("Vary"); !containsHeaderValue(vary, "HX-Request") {
		t.Fatalf("HTMX response Vary = %q", vary)
	}
}

func TestDocsHTMXErrorStaysOutOfTheSwapTarget(t *testing.T) {
	handler := newDocsTestHandler(t)
	response := serveDocs(t, handler, true, "/docs/missing")

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if strings.Contains(strings.ToLower(response.Body.String()), "<!doctype html>") {
		t.Fatal("HTMX error response contains a full page")
	}
	if vary := response.Header().Values("Vary"); !containsHeaderValue(vary, "HX-Request") {
		t.Fatalf("error response Vary = %q", vary)
	}
}

func TestDocsServesMarkdownSource(t *testing.T) {
	handler := newDocsTestHandler(t)
	response := serveDocs(t, handler, false, "/docs.md")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/markdown; charset=utf-8" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if !strings.Contains(response.Body.String(), "## First step") {
		t.Fatalf("markdown response = %q", response.Body.String())
	}
}

func newDocsTestHandler(t *testing.T) *DocsHandler {
	t.Helper()
	contentPath := t.TempDir()
	docsPath := filepath.Join(contentPath, "docs")
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		t.Fatalf("create docs directory: %v", err)
	}
	page := "---\ntitle: Introduction\ndescription: Start here\norder: 1\n---\n\n## First step\n\nHello.\n"
	if err := os.WriteFile(filepath.Join(docsPath, "introduction.md"), []byte(page), 0o644); err != nil {
		t.Fatalf("write docs page: %v", err)
	}
	return NewDocsHandler(service.NewDocsService(contentPath))
}

func serveDocs(t *testing.T, handler *DocsHandler, action bool, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if action {
		request.Header.Set("HX-Request", "true")
	}
	ctx := ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{
		AppURL:       "https://example.com",
		SupportEmail: "support@example.com",
	})
	ctx = ctxkeys.WithURLPath(ctx, request.URL.Path)
	ctx = ctxkeys.WithCSRFToken(ctx, "csrf-token")
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ShowDocs(recorder, request)
	return recorder
}

func containsHeaderValue(values []string, want string) bool {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.TrimSpace(part) == want {
				return true
			}
		}
	}
	return false
}
