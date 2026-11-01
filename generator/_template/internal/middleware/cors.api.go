package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, POST, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders = "Authorization, Content-Type, X-CSRF-Token"
	corsExposedHeaders = "Retry-After"
	corsMaxAge         = "600"
)

// CORS lets the listed origins call the JSON API from a browser, with
// cookies or a bearer token. It applies only to paths under /api/. Origins
// match exactly, and an origin that is not listed gets no CORS headers, so
// the browser blocks the response. Preflight requests under /api/ are
// answered here with 204 and never reach the router. Listed origins may read
// Retry-After, so a frontend can show the wait after a 429.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			_, listed := allowed[origin]
			listed = listed && origin != ""
			if listed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Expose-Headers", corsExposedHeaders)
			}

			isPreflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
			if !isPreflight {
				next.ServeHTTP(w, r)
				return
			}
			if listed {
				w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
				w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
