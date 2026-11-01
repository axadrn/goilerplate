package middleware

import "testing"

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
