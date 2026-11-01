package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/layouts"
	"{{.ModulePath}}/internal/ui/pages"
)

type WorkspaceHandler struct {
	workspaces *service.WorkspaceService
}

func NewWorkspaceHandler(workspaces *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

func (h *WorkspaceHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	h.redirectToFirstWorkspace(w, r, user.ID)
}

func (h *WorkspaceHandler) Page(w http.ResponseWriter, r *http.Request) {
	workspace := ctxkeys.Workspace(r.Context())
	member := ctxkeys.WorkspaceMember(r.Context())
	members, err := h.workspaces.Members(member)
	if err != nil {
		slog.Error("workspace members could not be loaded", "error", err, "workspace_id", workspace.ID)
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Workspace unavailable", "The workspace members could not be loaded. Please try again."))
		return
	}
	var invitations []model.WorkspaceInvitation
	if member.Can(model.WorkspacePermissionManageMembers) {
		invitations, err = h.workspaces.PendingInvitations(workspace.ID)
		if err != nil {
			slog.Error("invitations could not be loaded", "error", err, "workspace_id", workspace.ID)
			w.WriteHeader(http.StatusInternalServerError)
			render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Workspace unavailable", "The invitations could not be loaded. Please try again."))
			return
		}
	}
	render(w, r, pages.Workspace(members, invitations))
}

func (h *WorkspaceHandler) ActivityPage(w http.ResponseWriter, r *http.Request) {
	member := ctxkeys.WorkspaceMember(r.Context())
	if !member.Can(model.WorkspacePermissionViewAuditLog) {
		w.WriteHeader(http.StatusForbidden)
		render(w, r, pages.ErrorPage(http.StatusForbidden, "Access denied", "You do not have permission to view workspace activity."))
		return
	}
	events, err := h.workspaces.AuditEvents(member.WorkspaceID)
	if err != nil {
		slog.Error("activity could not be loaded", "error", err, "workspace_id", member.WorkspaceID)
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Activity unavailable", "Workspace activity could not be loaded. Please try again."))
		return
	}
	render(w, r, pages.WorkspaceActivity(events))
}

func (h *WorkspaceHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	if err := h.workspaces.UpdateName(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()), r.FormValue("name")); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.renderWorkspaceAction(w, r, "Workspace name updated.")
}

func (h *WorkspaceHandler) Invite(w http.ResponseWriter, r *http.Request) {
	if err := h.workspaces.Invite(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()), r.FormValue("email"), r.FormValue("role")); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.renderWorkspaceAction(w, r, "Invitation sent.")
}

func (h *WorkspaceHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	if err := h.workspaces.ResendInvitation(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()), r.PathValue("invitationID")); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.renderWorkspaceAction(w, r, "Invitation sent again.")
}

func (h *WorkspaceHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspace := ctxkeys.Workspace(r.Context())
	if err := h.workspaces.RevokeInvitation(ctxkeys.WorkspaceMember(r.Context()), workspace.ID, r.PathValue("invitationID")); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.renderWorkspaceAction(w, r, "Invitation revoked.")
}

func (h *WorkspaceHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	member := ctxkeys.WorkspaceMember(r.Context())
	if err := h.workspaces.ChangeRole(member, member.WorkspaceID, r.PathValue("memberID"), r.FormValue("role")); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.renderWorkspaceAction(w, r, "Member role updated.")
}

func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	member := ctxkeys.WorkspaceMember(r.Context())
	memberID := r.PathValue("memberID")
	if err := h.workspaces.RemoveMember(member, member.WorkspaceID, memberID); err != nil {
		h.handleError(w, r, err)
		return
	}
	if member.ID == memberID {
		h.redirectToFirstWorkspace(w, r, member.UserID)
		return
	}
	h.renderWorkspaceAction(w, r, "Member removed.")
}

func (h *WorkspaceHandler) Leave(w http.ResponseWriter, r *http.Request) {
	workspace := ctxkeys.Workspace(r.Context())
	user := ctxkeys.User(r.Context())
	if err := h.workspaces.Leave(workspace.ID, user.ID); err != nil {
		h.handleError(w, r, err)
		return
	}
	h.redirectToFirstWorkspace(w, r, user.ID)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	if err := h.workspaces.Delete(ctxkeys.WorkspaceMember(r.Context()), workspace); err != nil {
		h.handleError(w, r, err)
		return
	}
	slog.Info("workspace deleted", "workspace_id", workspace.ID, "user_id", user.ID)
	h.redirectToFirstWorkspace(w, r, user.ID)
}

func (h *WorkspaceHandler) NewPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.NewWorkspace(""))
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace, err := h.workspaces.Create(user.ID, r.FormValue("name"))
	if err != nil {
		render(w, r, pages.NewWorkspaceForm("Enter a workspace name"))
		return
	}
	redirect(w, r, fmt.Sprintf("/app/%s/dashboard", workspace.Slug), http.StatusSeeOther)
}

func (h *WorkspaceHandler) redirectToWorkspace(w http.ResponseWriter, r *http.Request) {
	redirect(w, r, fmt.Sprintf("/app/%s/workspace", ctxkeys.Workspace(r.Context()).Slug), http.StatusSeeOther)
}

func (h *WorkspaceHandler) renderWorkspaceAction(w http.ResponseWriter, r *http.Request, description string) {
	if !isAction(r) {
		h.redirectToWorkspace(w, r)
		return
	}

	user := ctxkeys.User(r.Context())
	current := ctxkeys.Workspace(r.Context())
	workspace, member, err := h.workspaces.BySlugForUser(current.Slug, user.ID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	members, err := h.workspaces.Members(member)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	var invitations []model.WorkspaceInvitation
	if member.Can(model.WorkspacePermissionManageMembers) {
		invitations, err = h.workspaces.PendingInvitations(workspace.ID)
		if err != nil {
			h.handleError(w, r, err)
			return
		}
	}

	workspaces, err := h.workspaces.ListForUser(user.ID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	ctx := ctxkeys.WithWorkspace(r.Context(), workspace)
	ctx = ctxkeys.WithWorkspaceMember(ctx, member)
	ctx = ctxkeys.WithWorkspaces(ctx, workspaces)
	r = r.WithContext(ctx)

	renderOOB(w, r, toast.Toast(toast.Props{Title: "Workspace updated", Description: description, Type: toast.TypeSuccess}), "beforeend:#toast-container")
	render(w, r, pages.WorkspaceContent(members, invitations))
	render(w, r, layouts.WorkspaceIdentity(workspace, workspaces))
}

func (h *WorkspaceHandler) redirectToFirstWorkspace(w http.ResponseWriter, r *http.Request, userID string) {
	members, err := h.workspaces.ListForUser(userID)
	if err != nil || len(members) == 0 {
		redirect(w, r, "/workspaces/new", http.StatusSeeOther)
		return
	}
	redirect(w, r, fmt.Sprintf("/app/%s/dashboard", members[0].Slug), http.StatusSeeOther)
}

func (h *WorkspaceHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	title := "Action failed"
	description := "The workspace action could not be completed. Please try again."
	switch {
	case errors.Is(err, service.ErrWorkspacePermission):
		status = http.StatusForbidden
		title = "Action not allowed"
		description = "You do not have permission to perform this workspace action."
	case errors.Is(err, repository.ErrLastOwner):
		status = http.StatusUnprocessableEntity
		title = "Owner required"
		description = "A workspace must always have at least one owner."
	case errors.Is(err, repository.ErrSeatLimit):
		status = http.StatusUnprocessableEntity
		title = "No seats available"
		description = "Upgrade the workspace or remove a member before inviting someone new."
	case errors.Is(err, repository.ErrAlreadyMember):
		status = http.StatusUnprocessableEntity
		title = "Already a member"
		description = "That user already belongs to this workspace."
	case errors.Is(err, service.ErrInvalidRole):
		status = http.StatusUnprocessableEntity
		title = "Invalid role"
		description = "Choose Owner, Admin, or Member where allowed."
	case errors.Is(err, service.ErrInvalidWorkspace):
		status = http.StatusUnprocessableEntity
		title = "Invalid workspace name"
		description = "Workspace names must contain between 2 and 80 printable characters."
	case errors.Is(err, service.ErrInvalidInviteEmail):
		status = http.StatusUnprocessableEntity
		title = "Invalid email"
		description = "Enter a valid email address."
	case errors.Is(err, service.ErrActiveSubscription):
		status = http.StatusUnprocessableEntity
		title = "Workspace cannot be deleted"
		description = "Finish or wait for checkout to expire. Cancel paid subscriptions and wait for access to end before deleting this workspace."
	case errors.Is(err, repository.ErrWorkspaceMemberNotFound), errors.Is(err, repository.ErrWorkspaceInvitationNotFound):
		status = http.StatusNotFound
		title = "Not found"
		description = "The requested member or invitation no longer exists."
	default:
		slog.Error("workspace action failed", "error", err)
	}

	if isAction(r) {
		renderOOB(w, r, toast.Toast(toast.Props{Title: title, Description: description, Type: toast.TypeError}), "beforeend:#toast-container")
		return
	}
	w.WriteHeader(status)
	render(w, r, pages.ErrorPage(status, title, description))
}
