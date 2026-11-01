package api

import (
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

// TokenAccess says what a personal API token may do on a route. Session
// callers are never limited by it.
type TokenAccess int

const (
	// TokenDenied answers 403 forbidden to token callers. It is the zero
	// value, so a new route stays closed to tokens until it opts in.
	TokenDenied TokenAccess = iota
	// TokenRead lets tokens with the read scope call the route.
	TokenRead
	// TokenWrite lets only tokens with the write scope call the route.
	TokenWrite
)

// RequireScopeAPI answers 403 insufficient_scope when a token caller lacks
// scope. Session and guest requests pass.
func RequireScopeAPI(scope string) Wrap {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := ctxkeys.APIToken(r.Context())
			if token != nil && !token.HasScope(scope) {
				Error(w, http.StatusForbidden, "insufficient_scope", "This API token needs the "+scope+" scope.")
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// DenyAPITokens answers 403 forbidden to token callers. Account security,
// tokens themselves, deletions, billing and admin need a signed-in session.
func DenyAPITokens(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ctxkeys.APIToken(r.Context()) != nil {
			Error(w, http.StatusForbidden, "forbidden", "API tokens cannot use this endpoint. Sign in to continue.")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (access TokenAccess) wrap(next http.HandlerFunc) http.HandlerFunc {
	switch access {
	case TokenRead:
		return RequireScopeAPI(model.APITokenScopeRead)(next)
	case TokenWrite:
		return RequireScopeAPI(model.APITokenScopeWrite)(next)
	default:
		return DenyAPITokens(next)
	}
}
