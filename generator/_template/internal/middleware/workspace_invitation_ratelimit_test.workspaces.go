package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestWorkspaceInvitationRateLimitIsScopedByManagerAndWorkspace(t *testing.T) {
	limiter := NewRateLimiter(2, time.Hour)
	t.Cleanup(limiter.Close)
	handler := RateLimitWorkspaceInvitations(limiter)(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := func(workspaceID, userID string) int {
		req := httptest.NewRequest(http.MethodPost, "/app/workspace/workspace/invitations", nil)
		member := &model.WorkspaceMember{WorkspaceID: workspaceID, UserID: userID, Role: model.WorkspaceRoleAdmin}
		req = req.WithContext(ctxkeys.WithWorkspaceMember(req.Context(), member))
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		return recorder.Code
	}

	if request("workspace-1", "manager-1") != http.StatusNoContent || request("workspace-1", "manager-1") != http.StatusNoContent {
		t.Fatal("invitation limiter rejected a request below the limit")
	}
	if status := request("workspace-1", "manager-1"); status != http.StatusTooManyRequests {
		t.Fatalf("third invitation status = %d, want %d", status, http.StatusTooManyRequests)
	}
	if status := request("workspace-1", "manager-2"); status != http.StatusNoContent {
		t.Fatalf("different manager status = %d, want %d", status, http.StatusNoContent)
	}
	if status := request("workspace-2", "manager-1"); status != http.StatusNoContent {
		t.Fatalf("different workspace status = %d, want %d", status, http.StatusNoContent)
	}
}
