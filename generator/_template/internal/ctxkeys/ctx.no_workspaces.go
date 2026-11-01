package ctxkeys

import (
	"context"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/model"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	UserKey         contextKey = "user"
	SubscriptionKey contextKey = "subscription"
	SessionKey      contextKey = "session"
	URLPathKey      contextKey = "url_path"
	RenderConfigKey contextKey = "render_config"
	CSRFTokenKey    contextKey = "csrf_token"
	AdminKey        contextKey = "admin"
)

func User(ctx context.Context) *model.User {
	user, _ := ctx.Value(UserKey).(*model.User)
	return user
}

func URLPath(ctx context.Context) string {
	path, _ := ctx.Value(URLPathKey).(string)
	return path
}

func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

func WithURLPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, URLPathKey, path)
}

func RenderConfig(ctx context.Context) config.RenderConfig {
	cfg, _ := ctx.Value(RenderConfigKey).(config.RenderConfig)
	return cfg
}

func WithRenderConfig(ctx context.Context, cfg config.RenderConfig) context.Context {
	return context.WithValue(ctx, RenderConfigKey, cfg)
}

func Subscription(ctx context.Context) *model.Subscription {
	subscription, _ := ctx.Value(SubscriptionKey).(*model.Subscription)
	return subscription
}

func WithSubscription(ctx context.Context, subscription *model.Subscription) context.Context {
	return context.WithValue(ctx, SubscriptionKey, subscription)
}

func Session(ctx context.Context) *model.Session {
	session, _ := ctx.Value(SessionKey).(*model.Session)
	return session
}

func WithSession(ctx context.Context, session *model.Session) context.Context {
	return context.WithValue(ctx, SessionKey, session)
}

func IsAdmin(ctx context.Context) bool {
	isAdmin, _ := ctx.Value(AdminKey).(bool)
	return isAdmin
}

func WithAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, AdminKey, isAdmin)
}

func CSRFToken(ctx context.Context) string {
	token, _ := ctx.Value(CSRFTokenKey).(string)
	return token
}

func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, CSRFTokenKey, token)
}
