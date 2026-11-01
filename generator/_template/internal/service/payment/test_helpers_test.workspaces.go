package payment

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

func createPaymentTestWorkspace(t *testing.T, database *sqlx.DB, ownerID, suffix string) (*model.Workspace, *model.Subscription) {
	t.Helper()
	now := time.Now()
	workspace := &model.Workspace{ID: suffix, Name: suffix, Slug: suffix, CreatedBy: &ownerID, CreatedAt: now, UpdatedAt: now}
	member := &model.WorkspaceMember{ID: "member-" + suffix, WorkspaceID: workspace.ID, UserID: ownerID, Role: model.WorkspaceRoleOwner, CreatedAt: now}
	subscription := &model.Subscription{ID: "subscription-" + suffix, WorkspaceID: workspace.ID, PlanID: model.SubscriptionPlanFree, Status: model.SubscriptionStatusActive, CreatedAt: now, UpdatedAt: now}
	audit := &model.WorkspaceAuditEvent{ID: "audit-" + suffix, WorkspaceID: workspace.ID, ActorUserID: &ownerID, Action: "workspace.created", TargetType: "workspace", TargetID: workspace.ID, Metadata: "{}", CreatedAt: now}
	if err := repository.NewWorkspaceRepository(database).CreateWithOwner(workspace, member, subscription, audit); err != nil {
		t.Fatalf("create additional workspace: %v", err)
	}
	return workspace, subscription
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
