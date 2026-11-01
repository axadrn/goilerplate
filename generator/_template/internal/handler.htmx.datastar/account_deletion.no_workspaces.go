package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
)

func accountDeletionError(w http.ResponseWriter, r *http.Request, userID string, err error) {
	if errors.Is(err, service.ErrActiveSubscription) {
		slog.Warn("account deletion failed: active subscription", "error", err, "user_id", userID)
		accountToast(w, r, toast.Toast(toast.Props{Title: "Active Subscription", Description: "Please cancel your subscription before deleting your account. Visit billing to manage your subscription.", Type: toast.TypeError}))
		return
	}
	slog.Error("account deletion failed", "error", err, "user_id", userID)
	accountToast(w, r, toast.Toast(toast.Props{Title: "Error", Description: "Failed to delete account. Please try again.", Type: toast.TypeError}))
}
