package api

import (
	"log/slog"
	"net/http"
	"time"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/service"
)

// SessionsHandler lists and revokes the sessions of the signed-in user.
type SessionsHandler struct {
	authService *service.AuthService
}

func NewSessionsHandler(authService *service.AuthService) *SessionsHandler {
	return &SessionsHandler{authService: authService}
}

// Session is the public JSON shape of a session. It never carries the token
// hash or the raw user agent.
type Session struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Current     bool      `json:"current"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// SessionList is the answer of GET /api/v1/account/sessions.
type SessionList struct {
	Items []Session `json:"items"`
}

// List returns every active session and marks the current one.
func (h *SessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	sessions, err := h.authService.Sessions(user.ID)
	if err != nil {
		slog.Error("failed to list sessions", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}
	currentID := currentSessionID(r)
	items := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, newSession(session, currentID))
	}
	JSON(w, http.StatusOK, SessionList{Items: items})
}

// Revoke signs out one session. Revoking the current session also clears the
// cookie of a cookie request.
func (h *SessionsHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	sessionID := r.PathValue("id")
	err := h.authService.RevokeSession(user.ID, sessionID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	if sessionID == currentSessionID(r) && !ctxkeys.IsBearer(r.Context()) {
		h.authService.ClearSessionCookie(w)
	}
	NoContent(w)
}

// RevokeOthers signs out every session except the current one.
func (h *SessionsHandler) RevokeOthers(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	err := h.authService.RevokeOtherSessions(user.ID, currentSessionID(r))
	if err != nil {
		slog.Error("failed to revoke other sessions", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}
	NoContent(w)
}

func currentSessionID(r *http.Request) string {
	session := ctxkeys.Session(r.Context())
	if session == nil {
		return ""
	}
	return session.ID
}

func newSession(session model.Session, currentID string) Session {
	return Session{
		ID:          session.ID,
		DisplayName: session.DisplayName(),
		Current:     session.ID == currentID,
		LastSeenAt:  session.LastSeenAt.UTC(),
		CreatedAt:   session.CreatedAt.UTC(),
	}
}
