package model

import (
	"testing"
)

func TestMembershipCapabilities(t *testing.T) {
	tests := []struct {
		role       string
		permission string
		want       bool
	}{
		{WorkspaceRoleOwner, WorkspacePermissionManageBilling, true},
		{WorkspaceRoleOwner, WorkspacePermissionManageOwners, true},
		{WorkspaceRoleAdmin, WorkspacePermissionManageWorkspace, true},
		{WorkspaceRoleAdmin, WorkspacePermissionManageMembers, true},
		{WorkspaceRoleAdmin, WorkspacePermissionViewAuditLog, true},
		{WorkspaceRoleAdmin, WorkspacePermissionManageBilling, false},
		{WorkspaceRoleAdmin, WorkspacePermissionManageOwners, false},
		{WorkspaceRoleMember, WorkspacePermissionManageMembers, false},
		{WorkspaceRoleMember, WorkspacePermissionViewAuditLog, false},
	}

	for _, test := range tests {
		member := &WorkspaceMember{Role: test.role}
		if got := member.Can(test.permission); got != test.want {
			t.Fatalf("role %q permission %q = %t, want %t", test.role, test.permission, got, test.want)
		}
	}

	var missing *WorkspaceMember
	if missing.Can(WorkspacePermissionManageMembers) {
		t.Fatal("nil member has permissions")
	}
}

func TestPlanPriceFormatting(t *testing.T) {
	if FormatPlanPrice(2900) != "$29" || FormatPlanPrice(2999) != "$29.99" {
		t.Fatalf("plan price formatting is inconsistent")
	}
}
