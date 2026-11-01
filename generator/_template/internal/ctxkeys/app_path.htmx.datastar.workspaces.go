package ctxkeys

import "context"

func AppBasePath(ctx context.Context) string {
	return "/app/" + Workspace(ctx).Slug
}
