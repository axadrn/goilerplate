package api

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
)

const (
	// DefaultActivityLimit is the page size of workspace activity.
	DefaultActivityLimit = 50
	// MaxActivityLimit is the largest page of workspace activity.
	MaxActivityLimit = 200
	// ActivityLimitMessage explains an invalid activity limit.
	ActivityLimitMessage = "Limit must be a number between 1 and 200."
)

var (
	// ErrInvalidCursor is returned for an activity cursor that does not
	// decode or points to an event that no longer exists.
	ErrInvalidCursor = errors.New("invalid activity cursor")
	// ErrActivityPermission is returned to members who may not view workspace
	// activity.
	ErrActivityPermission = errors.New("workspace activity permission required")
)

// WorkspaceHandler serves workspaces, their members and invitations, and
// workspace activity. Every rule and permission check lives in
// WorkspaceService, like for the HTML handler.
type WorkspaceHandler struct {
	workspaces *service.WorkspaceService
}

func NewWorkspaceHandler(workspaces *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

// WorkspaceMember is a member of a workspace. Email is empty for viewers
// who may not manage members.
type WorkspaceMember struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// NewWorkspaceMember converts a member model into its public JSON shape.
func NewWorkspaceMember(member *model.WorkspaceMember) WorkspaceMember {
	return WorkspaceMember{
		ID:        member.ID,
		UserID:    member.UserID,
		Name:      member.Name,
		Email:     member.Email,
		Role:      member.Role,
		CreatedAt: member.CreatedAt.UTC(),
	}
}

// PendingInvitation is an open invitation of a workspace.
type PendingInvitation struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// WorkspacePermissions lists what the current member may do.
type WorkspacePermissions struct {
	ManageWorkspace bool `json:"manage_workspace"`
	ManageMembers   bool `json:"manage_members"`
	ManageOwners    bool `json:"manage_owners"`
	ManageBilling   bool `json:"manage_billing"`
	DeleteWorkspace bool `json:"delete_workspace"`
	ViewAuditLog    bool `json:"view_audit_log"`
}

// Seats compares the member count with the plan limit. Limit is null when
// the plan has no member limit.
type Seats struct {
	Used  int  `json:"used"`
	Limit *int `json:"limit"`
}

// WorkspaceDetail is a workspace with the role and permissions of the
// current member, its members, pending invitations and seats.
type WorkspaceDetail struct {
	Workspace   Workspace            `json:"workspace"`
	Role        string               `json:"role"`
	Permissions WorkspacePermissions `json:"permissions"`
	Members     []WorkspaceMember    `json:"members"`
	Invitations []PendingInvitation  `json:"invitations"`
	Seats       Seats                `json:"seats"`
}

// WorkspaceEvent is one entry of the workspace activity.
type WorkspaceEvent struct {
	ID          string    `json:"id"`
	ActorUserID *string   `json:"actor_user_id"`
	ActorEmail  string    `json:"actor_email"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}

// ActivityPage is one page of workspace activity. NextCursor continues after
// the last event and is null on the last page.
type ActivityPage struct {
	Items      []WorkspaceEvent `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

type itemsResponse[T any] struct {
	Items []T `json:"items"`
}

type workspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

type memberResponse struct {
	Member WorkspaceMember `json:"member"`
}

type invitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type roleRequest struct {
	Role string `json:"role"`
}

// List returns the workspaces of the user with the user's role.
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	memberships, err := h.workspaces.ListForUser(ctxkeys.User(r.Context()).ID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, itemsResponse[Membership]{Items: NewMemberships(visibleMemberships(r, memberships))})
}

// Create creates a workspace owned by the user.
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request nameRequest
	if !Decode(w, r, &request) {
		return
	}
	workspace, err := h.workspaces.Create(user.ID, request.Name)
	if err != nil {
		ServiceError(w, err)
		return
	}
	slog.Info("workspace created", "workspace_id", workspace.ID, "user_id", user.ID)
	JSON(w, http.StatusCreated, Membership{Workspace: NewWorkspace(workspace), Role: model.WorkspaceRoleOwner})
}

// Show returns the workspace page data: the workspace, the role and
// permissions of the member, members, pending invitations for members who
// may manage them, and seats.
func (h *WorkspaceHandler) Show(w http.ResponseWriter, r *http.Request) {
	detail, err := NewWorkspaceDetail(h.workspaces, ctxkeys.Workspace(r.Context()), ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Subscription(r.Context()))
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, detail)
}

// NewWorkspaceDetail loads the members of a workspace, and its pending
// invitations for members who may manage them, and builds the detail the
// member sees. Seats.Limit comes from the plan of subscription.
func NewWorkspaceDetail(workspaces *service.WorkspaceService, workspace *model.Workspace, member *model.WorkspaceMember, subscription *model.Subscription) (WorkspaceDetail, error) {
	members, err := workspaces.Members(member)
	if err != nil {
		return WorkspaceDetail{}, err
	}
	invitations := []model.WorkspaceInvitation{}
	if member.Can(model.WorkspacePermissionManageMembers) {
		invitations, err = workspaces.PendingInvitations(workspace.ID)
		if err != nil {
			return WorkspaceDetail{}, err
		}
	}

	detail := WorkspaceDetail{
		Workspace:   NewWorkspace(workspace),
		Role:        member.Role,
		Permissions: newWorkspacePermissions(member),
		Members:     make([]WorkspaceMember, 0, len(members)),
		Invitations: make([]PendingInvitation, 0, len(invitations)),
		Seats:       Seats{Used: len(members)},
	}
	for index := range members {
		detail.Members = append(detail.Members, NewWorkspaceMember(&members[index]))
	}
	for _, invitation := range invitations {
		detail.Invitations = append(detail.Invitations, PendingInvitation{
			ID:        invitation.ID,
			Email:     invitation.Email,
			Role:      invitation.Role,
			ExpiresAt: invitation.ExpiresAt.UTC(),
			CreatedAt: invitation.CreatedAt.UTC(),
		})
	}
	if subscription != nil {
		limit := subscription.Access().Limit(access.LimitMembers)
		if limit != access.Unlimited {
			detail.Seats.Limit = &limit
		}
	}
	return detail, nil
}

// UpdateName renames the workspace. The slug stays.
func (h *WorkspaceHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	var request nameRequest
	if !Decode(w, r, &request) {
		return
	}
	err := h.workspaces.UpdateName(ctxkeys.WorkspaceMember(r.Context()), workspace, request.Name)
	if err != nil {
		ServiceError(w, err)
		return
	}
	updated, _, err := h.workspaces.BySlugForUser(workspace.Slug, user.ID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, workspaceResponse{Workspace: NewWorkspace(updated)})
}

// Invite sends an invitation. It answers 202 because the invitation is
// delivered by email; GET on the workspace lists it.
func (h *WorkspaceHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var request invitationRequest
	if !Decode(w, r, &request) {
		return
	}
	err := h.workspaces.Invite(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()), request.Email, request.Role)
	if err != nil {
		ServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// ResendInvitation sends a pending invitation again.
func (h *WorkspaceHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	err := h.workspaces.ResendInvitation(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()), r.PathValue("invitationID"))
	if err != nil {
		ServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// RevokeInvitation revokes a pending invitation.
func (h *WorkspaceHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	err := h.workspaces.RevokeInvitation(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Workspace(r.Context()).ID, r.PathValue("invitationID"))
	if err != nil {
		ServiceError(w, err)
		return
	}
	NoContent(w)
}

// ChangeRole changes the role of a member and returns the member.
func (h *WorkspaceHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	member := ctxkeys.WorkspaceMember(r.Context())
	memberID := r.PathValue("memberID")
	var request roleRequest
	if !Decode(w, r, &request) {
		return
	}
	err := h.workspaces.ChangeRole(member, member.WorkspaceID, memberID, request.Role)
	if err != nil {
		ServiceError(w, err)
		return
	}
	members, err := h.workspaces.Members(member)
	if err != nil {
		ServiceError(w, err)
		return
	}
	for index := range members {
		if members[index].ID == memberID {
			JSON(w, http.StatusOK, memberResponse{Member: NewWorkspaceMember(&members[index])})
			return
		}
	}
	ServiceError(w, repository.ErrWorkspaceMemberNotFound)
}

// RemoveMember removes a member from the workspace.
func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	member := ctxkeys.WorkspaceMember(r.Context())
	err := h.workspaces.RemoveMember(member, member.WorkspaceID, r.PathValue("memberID"))
	if err != nil {
		ServiceError(w, err)
		return
	}
	NoContent(w)
}

// Leave removes the user from the workspace.
func (h *WorkspaceHandler) Leave(w http.ResponseWriter, r *http.Request) {
	err := h.workspaces.Leave(ctxkeys.Workspace(r.Context()).ID, ctxkeys.User(r.Context()).ID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	NoContent(w)
}

// Delete deletes the workspace.
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	err := h.workspaces.Delete(ctxkeys.WorkspaceMember(r.Context()), workspace)
	if err != nil {
		ServiceError(w, err)
		return
	}
	slog.Info("workspace deleted", "workspace_id", workspace.ID, "user_id", user.ID)
	NoContent(w)
}

// Activity returns workspace activity, newest first, in pages of limit
// events (default 50, at most 200). next_cursor continues after the last
// event and is null on the last page.
func (h *WorkspaceHandler) Activity(w http.ResponseWriter, r *http.Request) {
	limit := DefaultActivityLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > MaxActivityLimit {
			ValidationError(w, map[string]string{"limit": ActivityLimitMessage})
			return
		}
		limit = parsed
	}
	page, err := NewActivityPage(h.workspaces, ctxkeys.WorkspaceMember(r.Context()), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, page)
}

// NewActivityPage loads up to limit events of the member's workspace, newest
// first, after the event cursor points to. An empty cursor starts with the
// newest event. Members who may not view the audit log get
// ErrActivityPermission. A cursor that does not decode or no longer exists
// returns ErrInvalidCursor.
func NewActivityPage(workspaces *service.WorkspaceService, member *model.WorkspaceMember, cursor string, limit int) (ActivityPage, error) {
	if !member.Can(model.WorkspacePermissionViewAuditLog) {
		return ActivityPage{}, ErrActivityPermission
	}
	afterID := ""
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(decoded) == 0 {
			return ActivityPage{}, ErrInvalidCursor
		}
		afterID = string(decoded)
	}

	events, next, err := workspaces.AuditEventsPage(member.WorkspaceID, afterID, limit)
	if errors.Is(err, repository.ErrAuditCursorNotFound) {
		return ActivityPage{}, ErrInvalidCursor
	}
	if err != nil {
		return ActivityPage{}, err
	}
	page := ActivityPage{Items: make([]WorkspaceEvent, 0, len(events))}
	for _, event := range events {
		page.Items = append(page.Items, WorkspaceEvent{
			ID:          event.ID,
			ActorUserID: event.ActorUserID,
			ActorEmail:  event.ActorEmail,
			Action:      event.Action,
			TargetType:  event.TargetType,
			TargetID:    event.TargetID,
			Metadata:    event.Metadata,
			CreatedAt:   event.CreatedAt.UTC(),
		})
	}
	if next != "" {
		nextCursor := base64.RawURLEncoding.EncodeToString([]byte(next))
		page.NextCursor = &nextCursor
	}
	return page, nil
}

func newWorkspacePermissions(member *model.WorkspaceMember) WorkspacePermissions {
	return WorkspacePermissions{
		ManageWorkspace: member.Can(model.WorkspacePermissionManageWorkspace),
		ManageMembers:   member.Can(model.WorkspacePermissionManageMembers),
		ManageOwners:    member.Can(model.WorkspacePermissionManageOwners),
		ManageBilling:   member.Can(model.WorkspacePermissionManageBilling),
		DeleteWorkspace: member.Can(model.WorkspacePermissionDeleteWorkspace),
		ViewAuditLog:    member.Can(model.WorkspacePermissionViewAuditLog),
	}
}
