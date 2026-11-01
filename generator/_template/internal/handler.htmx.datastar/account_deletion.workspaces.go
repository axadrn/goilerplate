package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
)

func accountDeletionError(w http.ResponseWriter, r *http.Request, userID string, err error) {
	if errors.Is(err, service.ErrSoleWorkspaceOwner) {
		slog.Warn("account deletion failed: sole workspace owner", "error", err, "user_id", userID)
		accountToast(w, r, toast.Toast(toast.Props{Title: "Workspace ownership required", Description: "Transfer ownership or delete workspaces where you are the only owner before deleting your account.", Type: toast.TypeError}))
		return
	}
	slog.Error("account deletion failed", "error", err, "user_id", userID)
	accountToast(w, r, toast.Toast(toast.Props{Title: "Error", Description: "Failed to delete account. Please try again.", Type: toast.TypeError}))
}
