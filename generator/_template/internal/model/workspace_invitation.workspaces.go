package model

import "time"

const WorkspaceInvitationRetention = 180 * 24 * time.Hour

type WorkspaceInvitation struct {
	ID            string     `db:"id"`
	WorkspaceID   string     `db:"workspace_id"`
	Email         string     `db:"email"`
	Role          string     `db:"role"`
	TokenHash     string     `db:"token_hash"`
	InvitedBy     string     `db:"invited_by"`
	ExpiresAt     time.Time  `db:"expires_at"`
	AcceptedAt    *time.Time `db:"accepted_at"`
	RevokedAt     *time.Time `db:"revoked_at"`
	CreatedAt     time.Time  `db:"created_at"`
	WorkspaceName string     `db:"workspace_name"`
	WorkspaceSlug string     `db:"workspace_slug"`
	InviterName   string     `db:"inviter_name"`
}
