package middleware

import (
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
)

// WithURLPath adds the current URL's path to the context
func WithURLPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxWithPath := ctxkeys.WithURLPath(r.Context(), r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctxWithPath))
	})
}
