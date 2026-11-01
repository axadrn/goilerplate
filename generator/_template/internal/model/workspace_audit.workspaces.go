package model

import "time"

type WorkspaceAuditEvent struct {
	ID          string    `db:"id"`
	WorkspaceID string    `db:"workspace_id"`
	ActorUserID *string   `db:"actor_user_id"`
	ActorEmail  string    `db:"actor_email"`
	Action      string    `db:"action"`
	TargetType  string    `db:"target_type"`
	TargetID    string    `db:"target_id"`
	Metadata    string    `db:"metadata"`
	CreatedAt   time.Time `db:"created_at"`
}

func WorkspaceAuditActionLabel(action string) string {
	switch action {
	case "workspace.created":
		return "Created the workspace"
	case "workspace.updated":
		return "Updated workspace settings"
	case "workspace.invitation.sent":
		return "Sent an invitation"
	case "workspace.invitation.created":
		return "Created an invitation"
	case "workspace.invitation.revoked":
		return "Revoked an invitation"
	case "workspace.invitation.accepted":
		return "Accepted an invitation"
	case "workspace.member.role_changed":
		return "Changed a member role"
	case "workspace.member.removed":
		return "Removed a member"
	case "workspace.member.left":
		return "Left the workspace"
	case "billing.checkout_started":
		return "Started a billing checkout"
	case "billing.management_opened":
		return "Opened billing management"
	default:
		return action
	}
}
