package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"{{.ModulePath}}/internal/brand"
	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestAdminDashboardRendersOperatorDataAndProtectedActions(t *testing.T) {
	now := time.Now()
	operatorID := "operator"
	workspaceID := "workspace-1"
	data := &model.AdminDashboard{
		Metrics: model.AdminMetrics{Users: 2, VerifiedUsers: 1, Workspaces: 2, PaidWorkspaces: 1, ActiveSessions: 3},
		Users: model.AdminUserPage{
			Users: []model.AdminUser{
				{ID: operatorID, Email: "operator@example.com", Name: "Operator", IsAdmin: true, CreatedAt: now},
				{ID: "target", Email: "target@example.com", Name: "Target", WorkspaceCount: 2, ActiveSessionCount: 1, CreatedAt: now},
			},
			Page:       1,
			TotalPages: 1,
			TotalUsers: 2,
		},
		Events: []model.AdminAuditEvent{{
			ID:          "event-1",
			ActorEmail:  "operator@example.com",
			WorkspaceID: &workspaceID,
			Action:      "workspace.deleted",
			TargetType:  "workspace",
			TargetID:    workspaceID,
			CreatedAt:   now,
		}},
	}
	ctx := context.Background()
	ctx = ctxkeys.WithRenderConfig(ctx, config.RenderConfig{})
	ctx = ctxkeys.WithURLPath(ctx, "/admin")
	ctx = ctxkeys.WithCSRFToken(ctx, "csrf-token")
	ctx = ctxkeys.WithUser(ctx, &model.User{ID: operatorID, Email: "operator@example.com"})

	var output bytes.Buffer
	if err := AdminDashboard(data).Render(ctx, &output); err != nil {
		t.Fatalf("render AdminDashboard(): %v", err)
	}
	html := output.String()
	for _, expected := range []string{"noindex, nofollow", brand.Name + " Admin", "target@example.com", "View as user", "Workspace deleted", "180 days"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("admin dashboard missing %q", expected)
		}
	}
	if strings.Contains(html, "/admin/users/operator/impersonate") {
		t.Fatal("admin dashboard allows self impersonation")
	}
	if !strings.Contains(html, "/admin/users/target/impersonate") || !strings.Contains(html, "csrf-token") {
		t.Fatal("admin dashboard omitted protected target action")
	}
}
