package pages

import (
	"context"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func projectPageContext(path string) context.Context {
	ctx := context.Background()
	ctx = ctxkeys.WithRenderConfig(ctx, config.RenderConfig{SupportEmail: "support@example.com"})
	ctx = ctxkeys.WithURLPath(ctx, path)
	ctx = ctxkeys.WithCSRFToken(ctx, "csrf-token")
	ctx = ctxkeys.WithUser(ctx, &model.User{ID: "user-1", Email: "alex@tinyrocket.test", Name: "Alex Vega"})
	ctx = ctxkeys.WithWorkspace(ctx, &model.Workspace{ID: "workspace-1", Name: "Tiny Rocket Co.", Slug: "tiny-rocket-co"})
	ctx = ctxkeys.WithWorkspaceMember(ctx, &model.WorkspaceMember{WorkspaceID: "workspace-1", UserID: "user-1", Role: model.WorkspaceRoleOwner})
	return ctx
}
