package ctxkeys

import "context"

const BearerTokenKey contextKey = "bearer_token"

// WithBearer marks a request as authenticated by an Authorization bearer
// token. No cookie is involved in such a request.
func WithBearer(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, BearerTokenKey, token)
}

// BearerToken returns the raw bearer token that authenticated the request, or
// an empty string when the request used a cookie or no session.
func BearerToken(ctx context.Context) string {
	token, _ := ctx.Value(BearerTokenKey).(string)
	return token
}

// IsBearer reports whether the request was authenticated by a bearer token.
func IsBearer(ctx context.Context) bool {
	return BearerToken(ctx) != ""
}
