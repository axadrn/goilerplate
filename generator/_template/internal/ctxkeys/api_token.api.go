package ctxkeys

import (
	"context"

	"{{.ModulePath}}/internal/model"
)

const APITokenKey contextKey = "api_token"

// WithAPIToken marks a request as authenticated by a personal API token. Such
// a request has a user but no session.
func WithAPIToken(ctx context.Context, token *model.APIToken) context.Context {
	return context.WithValue(ctx, APITokenKey, token)
}

// APIToken returns the personal API token that authenticated the request, or
// nil for session and guest requests.
func APIToken(ctx context.Context) *model.APIToken {
	token, _ := ctx.Value(APITokenKey).(*model.APIToken)
	return token
}
