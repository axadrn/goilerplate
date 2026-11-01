package api

import (
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
)

type csrfResponse struct {
	Token string `json:"token"`
}

// CSRFToken returns the CSRF token of the request for a browser frontend
// that signs in with the session cookie. CSRFProtection sets the csrf_token
// cookie on this GET when it is missing; the cookie stays HttpOnly, so the
// frontend sends this value as the X-CSRF-Token header instead.
func CSRFToken(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, csrfResponse{Token: ctxkeys.CSRFToken(r.Context())})
}
