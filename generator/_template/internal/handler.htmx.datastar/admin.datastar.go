package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
)

type AdminHandler struct {
	admin *service.AdminService
}

func NewAdminHandler(admin *service.AdminService) *AdminHandler {
	return &AdminHandler{admin: admin}
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	dashboard, err := h.admin.Dashboard(
		ctxkeys.User(r.Context()),
		ctxkeys.Session(r.Context()),
		r.URL.Query().Get("q"),
		page,
	)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	render(w, r, pages.AdminDashboard(dashboard))
}

func (h *AdminHandler) RevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	err := h.admin.RevokeUserSessions(ctxkeys.User(r.Context()), ctxkeys.Session(r.Context()), r.PathValue("userID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	query := r.FormValue("q")
	page, _ := strconv.Atoi(r.FormValue("page"))
	if !isAction(r) {
		values := url.Values{}
		if query != "" {
			values.Set("q", query)
		}
		values.Set("page", strconv.Itoa(page))
		redirect(w, r, "/admin?"+values.Encode(), http.StatusSeeOther)
		return
	}
	dashboard, err := h.admin.Dashboard(ctxkeys.User(r.Context()), ctxkeys.Session(r.Context()), query, page)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	appendElement(w, r, toast.Toast(toast.Props{Title: "Sessions revoked", Description: "The user has been signed out everywhere.", Type: toast.TypeSuccess}), "#toast-container")
	render(w, r, pages.AdminDashboardContent(dashboard))
}

func (h *AdminHandler) StartImpersonation(w http.ResponseWriter, r *http.Request) {
	err := h.admin.StartImpersonation(w, r, ctxkeys.User(r.Context()), ctxkeys.Session(r.Context()), r.PathValue("userID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	redirect(w, r, "/app", http.StatusSeeOther)
}

func (h *AdminHandler) StopImpersonation(w http.ResponseWriter, r *http.Request) {
	err := h.admin.StopImpersonation(w, ctxkeys.User(r.Context()), ctxkeys.Session(r.Context()))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	title := "Admin action failed"
	description := "The admin action could not be completed. Please try again."
	switch {
	case errors.Is(err, repository.ErrUserNotFound), errors.Is(err, repository.ErrSessionNotFound):
		status = http.StatusNotFound
		title = "User not found"
		description = "The requested user or session no longer exists."
	case errors.Is(err, service.ErrAdminPermission):
		status = http.StatusNotFound
		title = "Not found"
		description = "The requested admin resource was not found."
	case errors.Is(err, service.ErrAdminSelfAction), errors.Is(err, service.ErrAdminTargetIsOperator):
		status = http.StatusConflict
		title = "Action not allowed"
		description = err.Error()
	default:
		slog.Error("admin action failed", "error", err)
	}
	if isAction(r) {
		appendElement(w, r, toast.Toast(toast.Props{Title: title, Description: description, Type: toast.TypeError}), "#toast-container")
		return
	}
	w.WriteHeader(status)
	render(w, r, pages.ErrorPage(status, title, description))
}
