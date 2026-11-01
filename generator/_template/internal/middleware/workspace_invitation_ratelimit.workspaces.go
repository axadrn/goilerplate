package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

// NewWorkspaceInvitationLimiter returns the invitation budget: 30 emails an
// hour per manager and workspace. Create it once and pass it to the web routes,
// the JSON API and MCP, so every way to invite draws from the same budget.
func NewWorkspaceInvitationLimiter() *RateLimiter {
	return NewRateLimiter(30, time.Hour)
}

// RateLimitWorkspaceInvitations limits invitation email delivery per manager
// and workspace with the shared limiter.
func RateLimitWorkspaceInvitations(limiter *RateLimiter) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			member := ctxkeys.WorkspaceMember(r.Context())
			if member == nil || !member.Can(model.WorkspacePermissionManageMembers) {
				next(w, r)
				return
			}
			key := member.WorkspaceID + ":" + member.UserID
			if !limiter.Allow(key) {
				slog.Warn("invitation rate limit exceeded", "workspace_id", member.WorkspaceID, "user_id", member.UserID)
				writeRateLimitError(w, r, limiter)
				return
			}
			next(w, r)
		}
	}
}
