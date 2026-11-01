package ctxkeys

import "context"

func AppBasePath(context.Context) string {
	return "/app"
}
