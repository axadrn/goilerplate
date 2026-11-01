package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAllowedOrigin = "https://app.example.com"

func serveCORS(t *testing.T, request *http.Request) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	called := false
	handler := CORS([]string{testAllowedOrigin})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder, called
}

func TestCORSAllowsListedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/account", nil)
	request.Header.Set("Origin", testAllowedOrigin)

	recorder, called := serveCORS(t, request)

	if !called || recorder.Code != http.StatusOK {
		t.Fatalf("called = %t, status = %d", called, recorder.Code)
	}
	expectHeaders(t, recorder, map[string]string{
		"Access-Control-Allow-Origin":      testAllowedOrigin,
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Expose-Headers":    "Retry-After",
		"Vary":                             "Origin",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Max-Age":           "",
	})
}

func TestCORSExposesRetryAfterOnRateLimitedAnswers(t *testing.T) {
	handler := CORS([]string{testAllowedOrigin})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	request.Header.Set("Origin", testAllowedOrigin)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", recorder.Code)
	}
	expectHeaders(t, recorder, map[string]string{
		"Access-Control-Allow-Origin":   testAllowedOrigin,
		"Access-Control-Expose-Headers": "Retry-After",
		"Retry-After":                   "60",
	})
}

func TestCORSIgnoresUnlistedOrigins(t *testing.T) {
	for _, origin := range []string{"https://evil.example.com", "https://app.example.com.evil.com", "http://app.example.com", "null"} {
		t.Run(origin, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", nil)
			request.Header.Set("Origin", origin)

			recorder, called := serveCORS(t, request)

			if !called {
				t.Fatal("handler was not called")
			}
			expectNoCORSHeaders(t, recorder)
			got := recorder.Header().Get("Vary")
			if got != "Origin" {
				t.Fatalf("Vary = %q", got)
			}
		})
	}
}

func TestCORSWithoutOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/account", nil)

	recorder, called := serveCORS(t, request)

	if !called {
		t.Fatal("handler was not called")
	}
	expectNoCORSHeaders(t, recorder)
}

func TestCORSAnswersPreflightForListedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/account/name", nil)
	request.Header.Set("Origin", testAllowedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	request.Header.Set("Access-Control-Request-Headers", "content-type, x-csrf-token")

	recorder, called := serveCORS(t, request)

	if called {
		t.Fatal("preflight reached the router")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	expectHeaders(t, recorder, map[string]string{
		"Access-Control-Allow-Origin":      testAllowedOrigin,
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET, POST, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers":     "Authorization, Content-Type, X-CSRF-Token",
		"Access-Control-Max-Age":           "600",
		"Vary":                             "Origin",
	})
}

func TestCORSAnswersPreflightForUnlistedOriginWithoutHeaders(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/account/name", nil)
	request.Header.Set("Origin", "https://evil.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPatch)

	recorder, called := serveCORS(t, request)

	if called {
		t.Fatal("preflight reached the router")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	expectNoCORSHeaders(t, recorder)
}

func TestCORSSkipsPathsOutsideAPI(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		request := httptest.NewRequest(method, "/app/settings", nil)
		request.Header.Set("Origin", testAllowedOrigin)
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)

		recorder, called := serveCORS(t, request)

		if !called {
			t.Fatalf("%s: handler was not called", method)
		}
		expectNoCORSHeaders(t, recorder)
		got := recorder.Header().Get("Vary")
		if got != "" {
			t.Fatalf("%s: Vary = %q", method, got)
		}
	}
}

func TestCORSWithoutAllowedOrigins(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/account", nil)
	request.Header.Set("Origin", testAllowedOrigin)
	recorder := httptest.NewRecorder()
	CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(recorder, request)

	expectNoCORSHeaders(t, recorder)
}

func expectNoCORSHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for _, name := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
		"Access-Control-Expose-Headers",
	} {
		got := recorder.Header().Get(name)
		if got != "" {
			t.Fatalf("%s = %q, want none", name, got)
		}
	}
}

func expectHeaders(t *testing.T, recorder *httptest.ResponseRecorder, want map[string]string) {
	t.Helper()
	for name, value := range want {
		got := recorder.Header().Get(name)
		if got != value {
			t.Fatalf("%s = %q, want %q", name, got, value)
		}
	}
}
