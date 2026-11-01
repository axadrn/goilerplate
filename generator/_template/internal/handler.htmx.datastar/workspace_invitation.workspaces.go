package handler

import (
	"errors"
	"fmt"
	"net/http"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/pages"
)

type WorkspaceInvitationHandler struct {
	workspaces *service.WorkspaceService
	cfg        *config.Config
}

func NewWorkspaceInvitationHandler(workspaces *service.WorkspaceService, cfg *config.Config) *WorkspaceInvitationHandler {
	return &WorkspaceInvitationHandler{workspaces: workspaces, cfg: cfg}
}

func (h *WorkspaceInvitationHandler) Show(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	invitation, err := h.workspaces.WorkspaceInvitation(token)
	if err != nil {
		if invitationCookieShouldClear(err) {
			h.clearPendingCookie(w)
		}
		render(w, r, pages.WorkspaceInvitation(nil, invitationError(err)))
		return
	}
	user := ctxkeys.User(r.Context())
	if user == nil {
		h.setPendingCookie(w, token)
		redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	if user.Name == "" {
		h.setPendingCookie(w, token)
		redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
		return
	}
	render(w, r, pages.WorkspaceInvitation(invitation, ""))
}

func (h *WorkspaceInvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	token := r.PathValue("token")
	workspace, err := h.workspaces.AcceptInvitation(token, user)
	if err != nil {
		invitation, _ := h.workspaces.WorkspaceInvitation(token)
		if errors.Is(err, repository.ErrAlreadyMember) && invitation != nil {
			h.clearPendingCookie(w)
			redirect(w, r, fmt.Sprintf("/app/%s/dashboard", invitation.WorkspaceSlug), http.StatusSeeOther)
			return
		}
		if invitationCookieShouldClear(err) {
			h.clearPendingCookie(w)
		}
		render(w, r, pages.WorkspaceInvitation(invitation, invitationError(err)))
		return
	}
	h.clearPendingCookie(w)
	redirect(w, r, fmt.Sprintf("/app/%s/dashboard", workspace.Slug), http.StatusSeeOther)
}

func (h *WorkspaceInvitationHandler) setPendingCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "pending_invitation",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})
}

func (h *WorkspaceInvitationHandler) clearPendingCookie(w http.ResponseWriter) {
	clearPendingInvitationCookie(w, h.cfg)
}

func clearPendingInvitationCookie(w http.ResponseWriter, cfg *config.Config) {
	secure := cfg != nil && cfg.IsProduction()
	http.SetCookie(w, &http.Cookie{Name: "pending_invitation", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func invitationCookieShouldClear(err error) bool {
	return errors.Is(err, repository.ErrWorkspaceInvitationNotFound) ||
		errors.Is(err, repository.ErrWorkspaceInvitationExpired) ||
		errors.Is(err, repository.ErrWorkspaceInvitationRevoked) ||
		errors.Is(err, repository.ErrWorkspaceInvitationUsed) ||
		errors.Is(err, repository.ErrWorkspaceInvitationEmail) ||
		errors.Is(err, repository.ErrAlreadyMember) ||
		errors.Is(err, repository.ErrSeatLimit)
}

func invitationError(err error) string {
	switch {
	case errors.Is(err, repository.ErrWorkspaceInvitationExpired):
		return "This invitation has expired. Ask a workspace Owner or Admin to send a new one."
	case errors.Is(err, repository.ErrWorkspaceInvitationRevoked):
		return "This invitation was revoked."
	case errors.Is(err, repository.ErrWorkspaceInvitationUsed):
		return "This invitation has already been accepted."
	case errors.Is(err, repository.ErrWorkspaceInvitationEmail):
		return "Sign in with the email address that received this invitation."
	case errors.Is(err, repository.ErrSeatLimit):
		return "This workspace has no available seats. Ask an owner to upgrade or remove a member."
	case errors.Is(err, repository.ErrAlreadyMember):
		return "You already belong to this workspace."
	default:
		return "This invitation is invalid or no longer available."
	}
}
