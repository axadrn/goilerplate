package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMXCSPDoesNotAllowUnsafeEval(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	handler := NonceMiddleware(SecurityHeaders(SecurityHeadersConfig{}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	handler.ServeHTTP(response, request)

	if policy := response.Header().Get("Content-Security-Policy"); strings.Contains(policy, "'unsafe-eval'") {
		t.Fatalf("Content-Security-Policy = %q", policy)
	}
}
