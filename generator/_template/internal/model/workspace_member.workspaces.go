package model

import "time"

type WorkspaceMember struct {
	ID          string    `db:"id"`
	WorkspaceID string    `db:"workspace_id"`
	UserID      string    `db:"user_id"`
	Role        string    `db:"role"`
	Email       string    `db:"email"`
	Name        string    `db:"name"`
	CreatedAt   time.Time `db:"created_at"`
}

func (m *WorkspaceMember) Can(permission string) bool {
	if m == nil {
		return false
	}

	switch m.Role {
	case WorkspaceRoleOwner:
		return true
	case WorkspaceRoleAdmin:
		switch permission {
		case WorkspacePermissionManageWorkspace, WorkspacePermissionManageMembers, WorkspacePermissionViewAuditLog:
			return true
		}
	}

	return false
}

type WorkspaceMembership struct {
	Workspace
	Role string `db:"role"`
}
