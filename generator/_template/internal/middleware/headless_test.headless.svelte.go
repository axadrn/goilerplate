package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadlessSecurityHeadersNeedNoNonce(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/account", nil))

	policy := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "default-src 'none'") || !strings.Contains(policy, "frame-ancestors 'none'") {
		t.Fatalf("Content-Security-Policy = %q", policy)
	}
	if strings.Contains(policy, "nonce") || strings.Contains(policy, "script-src") {
		t.Fatalf("Content-Security-Policy = %q, want no script sources", policy)
	}
	for name, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		got := recorder.Header().Get(name)
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestHeadlessMiddlewareErrorsAreJSON(t *testing.T) {
	for _, path := range []string{"/", "/auth/google", "/api/v1/account"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			renderMiddlewareError(recorder, httptest.NewRequest(http.MethodPost, path, nil), http.StatusTooManyRequests, "Too many requests", "Please wait a moment and try again.")

			expectErrorEnvelope(t, recorder, http.StatusTooManyRequests, "rate_limited")
		})
	}
}

func TestHeadlessCSRFFailureIsJSON(t *testing.T) {
	handler := CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler was called without a CSRF token")
	}))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: "session"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	expectErrorEnvelope(t, recorder, http.StatusForbidden, "forbidden")
}

func expectErrorEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d", recorder.Code, status)
	}
	contentType := recorder.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	if body.Error.Code != code || body.Error.Message == "" {
		t.Fatalf("error = %+v, want code %q", body.Error, code)
	}
}
