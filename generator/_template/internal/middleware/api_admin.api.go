package middleware

import (
	"net/http"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/ctxkeys"
)

// RequireAdminAPI answers 401 unauthenticated without a session and 403
// forbidden for a user who is not an admin.
func RequireAdminAPI(next http.HandlerFunc) http.HandlerFunc {
	return RequireAuthAPI(func(w http.ResponseWriter, r *http.Request) {
		if !ctxkeys.IsAdmin(r.Context()) {
			api.Error(w, http.StatusForbidden, "forbidden", "Admin access is required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
