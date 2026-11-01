package model

import "time"

const AdminAuditRetention = 180 * 24 * time.Hour

type AdminMetrics struct {
	Users          int
	VerifiedUsers  int
	Workspaces     int
	PaidWorkspaces int
	ActiveSessions int
}

type AdminUser struct {
	ID                 string     `db:"id"`
	Email              string     `db:"email"`
	Name               string     `db:"name"`
	EmailVerifiedAt    *time.Time `db:"email_verified_at"`
	CreatedAt          time.Time  `db:"created_at"`
	WorkspaceCount     int        `db:"workspace_count"`
	ActiveSessionCount int        `db:"active_session_count"`
	IsAdmin            bool       `db:"-"`
}

type AdminUserPage struct {
	Users      []AdminUser
	Query      string
	Page       int
	TotalPages int
	TotalUsers int
}

type AdminDashboard struct {
	Metrics AdminMetrics
	Users   AdminUserPage
	Events  []AdminAuditEvent
}

type AdminAuditEvent struct {
	ID          string    `db:"id"`
	ActorUserID *string   `db:"actor_user_id"`
	ActorEmail  string    `db:"actor_email"`
	WorkspaceID *string   `db:"workspace_id"`
	Action      string    `db:"action"`
	TargetType  string    `db:"target_type"`
	TargetID    string    `db:"target_id"`
	Metadata    string    `db:"metadata"`
	CreatedAt   time.Time `db:"created_at"`
}

func AdminAuditActionLabel(action string) string {
	switch action {
	case "admin.impersonation_started":
		return "Impersonation started"
	case "admin.impersonation_stopped":
		return "Impersonation stopped"
	case "admin.sessions_revoked":
		return "User sessions revoked"
	case "workspace.deleted":
		return "Workspace deleted"
	case "subscription.updated":
		return "Subscription updated"
	default:
		return action
	}
}
