package middleware

import (
	"net/http"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
)

// WithWorkspaceAPI scopes an API route to the workspace in its path. It
// reuses WithWorkspace, whose errors are JSON envelopes for API requests, so
// an unknown or foreign workspace answers 404 not_found. A personal API token
// only reaches the workspace it is bound to; every other workspace answers the
// same 404.
func WithWorkspaceAPI(workspaceService *service.WorkspaceService, subscriptionService *service.SubscriptionService) func(http.HandlerFunc) http.HandlerFunc {
	withWorkspace := WithWorkspace(workspaceService, subscriptionService)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return withWorkspace(func(w http.ResponseWriter, r *http.Request) {
			token := ctxkeys.APIToken(r.Context())
			if token != nil && token.WorkspaceID != ctxkeys.Workspace(r.Context()).ID {
				api.Error(w, http.StatusNotFound, "not_found", "The requested workspace does not exist or you no longer have access.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
