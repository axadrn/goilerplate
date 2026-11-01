package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/app"
	"{{.ModulePath}}/internal/config"
)

// TestServerPathsStayOutOfTheApp sends server paths through the real router
// and checks that none of them reaches the SvelteKit fallback. Every answer of
// the app carries its script-src policy, built or not: index.html after pnpm
// build, the 503 before.
func TestServerPathsStayOutOfTheApp(t *testing.T) {
	handler := SetupRoutes(&app.App{Cfg: &config.Config{}})

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/app/settings", nil))
	if page.Code != http.StatusOK && page.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /app/settings status = %d, want the app page or the 503 of the unbuilt app", page.Code)
	}
	if !strings.Contains(page.Header().Get("Content-Security-Policy"), "script-src") {
		t.Fatalf("GET /app/settings Content-Security-Policy = %q, want the app policy", page.Header().Get("Content-Security-Policy"))
	}

	for _, request := range []struct {
		method string
		target string
	}{
		{http.MethodGet, "/api/v1/nope"},
		{http.MethodGet, "/api/v1/meta"},
		{http.MethodGet, "/api/v1/csrf"},
		{http.MethodPost, "/api/v1/nope"},
		{http.MethodGet, "/healthz"},
		{http.MethodPost, "/healthz"},
		{http.MethodGet, "/mcp"},
		{http.MethodGet, "/webhooks/payment"},
		{http.MethodGet, "/auth/google"},
		{http.MethodGet, "/auth/github/callback"},
	} {
		t.Run(request.method+" "+request.target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(request.method, request.target, nil))

			if recorder.Code == http.StatusServiceUnavailable || strings.Contains(recorder.Body.String(), "task build") {
				t.Fatalf("status = %d, body %q: the app answered a server path", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Header().Get("Content-Security-Policy"), "script-src") {
				t.Fatalf("Content-Security-Policy = %q, want the JSON policy", recorder.Header().Get("Content-Security-Policy"))
			}
		})
	}
}
