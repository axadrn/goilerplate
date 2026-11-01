package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/layouts"
)

type ProfileHandler struct {
	users *service.UserService
}

func NewProfileHandler(users *service.UserService) *ProfileHandler {
	return &ProfileHandler{users: users}
}

func (h *ProfileHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	name := r.FormValue("name")
	if strings.TrimSpace(name) == "" {
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Error", Description: "Name is required", Type: toast.TypeError}), "beforeend:#toast-container")
		return
	}
	updatedUser, err := h.users.UpdateName(user.ID, name)
	if err != nil {
		slog.Error("failed to update name", "error", err, "user_id", user.ID)
		renderOOB(w, r, toast.Toast(toast.Props{Title: "Error", Description: "Failed to update name", Type: toast.TypeError}), "beforeend:#toast-container")
		return
	}
	renderOOB(w, r, toast.Toast(toast.Props{Title: "Success", Description: "Name updated successfully", Type: toast.TypeSuccess}), "beforeend:#toast-container")
	render(w, r, layouts.SidebarIdentityUpdates(updatedUser))
}
