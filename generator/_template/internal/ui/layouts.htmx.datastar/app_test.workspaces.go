package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestAppLayoutAlwaysShowsWorkspaceMenu(t *testing.T) {
	oneWorkspace := renderAppLayout(t, model.WorkspaceRoleOwner, 1)
	for _, expected := range []string{"Switch workspace", "Current workspace", "Workspace settings", "New workspace", "Owner"} {
		if !strings.Contains(oneWorkspace, expected) {
			t.Fatalf("single-workspace menu does not contain %q", expected)
		}
	}

	twoWorkspaces := renderAppLayout(t, model.WorkspaceRoleOwner, 2)
	if !strings.Contains(twoWorkspaces, "Second Workspace") || !strings.Contains(twoWorkspaces, "Member") {
		t.Fatal("multi-workspace layout does not show both workspace choices")
	}
}

func TestAppLayoutHidesBillingFromNonOwners(t *testing.T) {
	owner := renderAppLayout(t, model.WorkspaceRoleOwner, 1)
	if !strings.Contains(owner, "Billing</span>") {
		t.Fatal("owner layout hides billing")
	}

	admin := renderAppLayout(t, model.WorkspaceRoleAdmin, 1)
	if strings.Contains(admin, "Billing</span>") {
		t.Fatal("admin layout shows owner-only billing")
	}
}

func TestAppLayoutShowsAdminLinkOnlyToOperators(t *testing.T) {
	ordinary := renderAppLayout(t, model.WorkspaceRoleOwner, 1)
	if strings.Contains(ordinary, `href="/admin"`) {
		t.Fatal("ordinary layout shows the Admin link")
	}

	admin := renderAppLayoutWithAdmin(t)
	if !strings.Contains(admin, `href="/admin"`) {
		t.Fatal("operator layout hides the Admin link")
	}
}

func TestBaseLayoutShowsReadOnlyImpersonationBanner(t *testing.T) {
	operatorID := "operator"
	ctx := context.Background()
	ctx = ctxkeys.WithRenderConfig(ctx, config.RenderConfig{})
	ctx = ctxkeys.WithCSRFToken(ctx, "csrf-token")
	ctx = ctxkeys.WithSession(ctx, &model.Session{ImpersonatorUserID: &operatorID})

	var output bytes.Buffer
	if err := Base().Render(ctx, &output); err != nil {
		t.Fatalf("render Base(): %v", err)
	}
	html := output.String()
	if !strings.Contains(html, "viewing the application as another user") || !strings.Contains(html, "/admin/impersonation/stop") || !strings.Contains(html, "csrf-token") {
		t.Fatal("impersonation banner is incomplete")
	}
}

func renderAppLayoutWithAdmin(t *testing.T) string {
	t.Helper()
	workspace := &model.Workspace{ID: "workspace-1", Name: "First Workspace", Slug: "first-workspace"}
	member := &model.WorkspaceMember{ID: "member-1", WorkspaceID: workspace.ID, UserID: "user-1", Role: model.WorkspaceRoleOwner}
	ctx := context.Background()
	ctx = ctxkeys.WithRenderConfig(ctx, config.RenderConfig{SupportEmail: "support@example.com"})
	ctx = ctxkeys.WithURLPath(ctx, "/app/first-workspace/dashboard")
	ctx = ctxkeys.WithUser(ctx, &model.User{ID: "user-1", Email: "user@example.com", Name: "Test User"})
	ctx = ctxkeys.WithWorkspace(ctx, workspace)
	ctx = ctxkeys.WithWorkspaceMember(ctx, member)
	ctx = ctxkeys.WithWorkspaces(ctx, []model.WorkspaceMembership{{Workspace: *workspace, Role: member.Role}})
	ctx = ctxkeys.WithAdmin(ctx, true)

	var output bytes.Buffer
	if err := App("Dashboard").Render(ctx, &output); err != nil {
		t.Fatalf("render App(): %v", err)
	}
	return output.String()
}

func renderAppLayout(t *testing.T, role string, workspaceCount int) string {
	t.Helper()
	workspace := &model.Workspace{ID: "workspace-1", Name: "First Workspace", Slug: "first-workspace"}
	member := &model.WorkspaceMember{ID: "member-1", WorkspaceID: workspace.ID, UserID: "user-1", Role: role}
	workspaces := []model.WorkspaceMembership{{Workspace: *workspace, Role: member.Role}}
	if workspaceCount > 1 {
		workspaces = append(workspaces, model.WorkspaceMembership{
			Workspace: model.Workspace{ID: "workspace-2", Name: "Second Workspace", Slug: "second-workspace"},
			Role:      model.WorkspaceRoleMember,
		})
	}

	ctx := context.Background()
	ctx = ctxkeys.WithRenderConfig(ctx, config.RenderConfig{SupportEmail: "support@example.com"})
	ctx = ctxkeys.WithURLPath(ctx, "/app/first-workspace/dashboard")
	ctx = ctxkeys.WithUser(ctx, &model.User{ID: "user-1", Email: "user@example.com", Name: "Test User"})
	ctx = ctxkeys.WithWorkspace(ctx, workspace)
	ctx = ctxkeys.WithWorkspaceMember(ctx, member)
	ctx = ctxkeys.WithWorkspaces(ctx, workspaces)

	var output bytes.Buffer
	if err := App("Dashboard").Render(ctx, &output); err != nil {
		t.Fatalf("render App(): %v", err)
	}
	return output.String()
}
