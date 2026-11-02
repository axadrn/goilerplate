package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSafeRequestLogPathRedactsTokens(t *testing.T) {
	tests := map[string]string{
		"/app/acme/dashboard":                   "/app/acme/dashboard",
		"/auth/magic-link/raw-secret":           "/auth/magic-link/:token",
		"/api/v1/invitations/raw-secret":        "/api/v1/invitations/:token",
		"/api/v1/invitations/raw-secret/accept": "/api/v1/invitations/:token/accept",
		"/auth/forgot-password/raw-secret":      "/auth/forgot-password/:token",
		"/auth/verify-email-change/raw-secret":  "/auth/verify-email-change/:token",
		"/workspaces/invitations/raw-secret":    "/workspaces/invitations/:token",
	}
	for path, want := range tests {
		if got := safeRequestLogPath(path); got != want {
			t.Fatalf("safeRequestLogPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestResponseWriterFlushSendsStatusOnce(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}
	if err := http.NewResponseController(writer).Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if !writer.written {
		t.Fatal("Flush() did not record the status")
	}
	if _, err := writer.Write([]byte("data: x\n\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if !recorder.Flushed || recorder.Code != http.StatusOK {
		t.Fatalf("recorder flushed=%v code=%d", recorder.Flushed, recorder.Code)
	}
}
