package api

import (
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service"
)

func accountDeletionError(w http.ResponseWriter, userID string, err error) {
	if errors.Is(err, service.ErrSoleWorkspaceOwner) {
		slog.Warn("account deletion failed: sole workspace owner", "error", err, "user_id", userID)
		Error(w, http.StatusConflict, "sole_workspace_owner", "Transfer ownership or delete workspaces where you are the only owner before deleting your account.")
		return
	}
	slog.Error("account deletion failed", "error", err, "user_id", userID)
	ServiceError(w, err)
}
