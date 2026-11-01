package model

import "time"

const (
	WorkspaceRoleOwner  = "owner"
	WorkspaceRoleAdmin  = "admin"
	WorkspaceRoleMember = "member"
)

func WorkspaceRoleLabel(role string) string {
	switch role {
	case WorkspaceRoleOwner:
		return "Owner"
	case WorkspaceRoleAdmin:
		return "Admin"
	case WorkspaceRoleMember:
		return "Member"
	default:
		return role
	}
}

const (
	WorkspacePermissionManageBilling   = "manage_billing"
	WorkspacePermissionManageWorkspace = "manage_workspace"
	WorkspacePermissionDeleteWorkspace = "delete_workspace"
	WorkspacePermissionManageMembers   = "manage_members"
	WorkspacePermissionManageOwners    = "manage_owners"
	WorkspacePermissionViewAuditLog    = "view_audit_log"
)

type Workspace struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	CreatedBy *string   `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
