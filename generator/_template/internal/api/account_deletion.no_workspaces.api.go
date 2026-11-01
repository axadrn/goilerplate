package api

import (
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service"
)

func accountDeletionError(w http.ResponseWriter, userID string, err error) {
	if errors.Is(err, service.ErrActiveSubscription) {
		slog.Warn("account deletion failed: active subscription", "error", err, "user_id", userID)
		Error(w, http.StatusConflict, "active_subscription", "Cancel your subscription before deleting your account.")
		return
	}
	slog.Error("account deletion failed", "error", err, "user_id", userID)
	ServiceError(w, err)
}
