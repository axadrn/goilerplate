package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestWorkspaceContentUsesViewerPermissionsForMemberActions(t *testing.T) {
	for _, test := range []struct {
		name        string
		viewerRole  string
		memberRole  string
		wantActions bool
	}{
		{name: "owner manages member", viewerRole: model.WorkspaceRoleOwner, memberRole: model.WorkspaceRoleMember, wantActions: true},
		{name: "member cannot manage owner", viewerRole: model.WorkspaceRoleMember, memberRole: model.WorkspaceRoleOwner, wantActions: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = ctxkeys.WithWorkspace(ctx, &model.Workspace{ID: "workspace-1", Name: "Acme", Slug: "acme"})
			ctx = ctxkeys.WithWorkspaceMember(ctx, &model.WorkspaceMember{ID: "viewer", WorkspaceID: "workspace-1", UserID: "viewer", Role: test.viewerRole})
			ctx = ctxkeys.WithSubscription(ctx, &model.Subscription{WorkspaceID: "workspace-1", PlanID: model.SubscriptionPlanEnterprise, Status: model.SubscriptionStatusActive})
			ctx = ctxkeys.WithCSRFToken(ctx, "csrf-token")
			members := []model.WorkspaceMember{{ID: "listed-member", WorkspaceID: "workspace-1", UserID: "listed-member", Name: "Listed Member", Role: test.memberRole}}

			var output bytes.Buffer
			if err := WorkspaceContent(members, nil).Render(ctx, &output); err != nil {
				t.Fatalf("render WorkspaceContent(): %v", err)
			}
			html := output.String()
			for _, action := range []string{
				"/app/acme/workspace/members/listed-member/role",
				"/app/acme/workspace/members/listed-member/remove",
			} {
				if got := strings.Contains(html, action); got != test.wantActions {
					t.Fatalf("action %q present = %v, want %v", action, got, test.wantActions)
				}
			}
		})
	}
}
