package api

import (
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/validation"
)

type emailRequest struct {
	Email string `json:"email"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type nameRequest struct {
	Name string `json:"name"`
}

type passwordLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Logout revokes the bearer session of the request, or the cookie session
// and its cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := ctxkeys.BearerToken(r.Context())
	if token != "" {
		err := h.authService.RevokeSessionByToken(token)
		if err != nil {
			ServiceError(w, err)
			return
		}
		NoContent(w)
		return
	}
	err := h.authService.EndSession(r)
	if err != nil {
		slog.Error("failed to revoke session during logout", "error", err)
		ServiceError(w, err)
		return
	}
	h.authService.ClearSessionCookie(w)
	NoContent(w)
}

// ForgotPassword sends a sign-in link that removes the password. It always
// answers 202 so the response never reveals whether an account exists.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var request emailRequest
	if !Decode(w, r, &request) {
		return
	}
	email := strings.TrimSpace(request.Email)
	err := validation.ValidateEmail(email)
	if err != nil {
		ValidationError(w, map[string]string{"email": FieldMessage(err)})
		return
	}

	err = h.authService.SendForgotPasswordLink(email)
	if err != nil {
		slog.Warn("forgot password link send failed", "error", err)
	}
	w.WriteHeader(http.StatusAccepted)
}

// VerifyForgotPassword consumes a forgot password token, ends every session,
// revokes every personal API token, removes the password, and returns the new
// session.
func (h *AuthHandler) VerifyForgotPassword(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
	if !Decode(w, r, &request) {
		return
	}

	user, err := h.authService.VerifyMagicLink(request.Token)
	if err != nil {
		slog.Warn("forgot password verification failed", "error", err)
		invalidToken(w)
		return
	}

	err = h.authService.RevokeAllAccess(user.ID)
	if err != nil {
		slog.Error("failed to revoke access during forgot password flow", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	if user.HasPassword() {
		err = h.authService.RemovePassword(user.ID)
		if err != nil {
			slog.Error("failed to remove password during forgot password flow", "error", err, "user_id", user.ID)
			ServiceError(w, err)
			return
		}
		slog.Info("password removed via forgot password flow", "user_id", user.ID)
	}

	session, token, err := h.authService.CreateFreshSession(r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	slog.Info("user logged in via forgot password flow", "user_id", user.ID)
	h.writeSession(w, r, session, token)
}

// VerifyEmailChange consumes an email change token, moves the pending
// address to the account, replaces every session of the user like the web
// flow, and returns the new session. A cookie caller also gets the new
// session cookie.
func (h *AuthHandler) VerifyEmailChange(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
	if !Decode(w, r, &request) {
		return
	}

	user, err := h.authService.VerifyEmailChange(request.Token)
	if err != nil {
		slog.Warn("email change verification failed", "error", err)
		invalidToken(w)
		return
	}

	session, token, err := h.authService.CreateFreshSession(r, user.ID)
	if err != nil {
		slog.Error("failed to create session after email change", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}
	if isCookieCaller(r) && !requestsCookieSession(r) {
		h.authService.SetSessionCookie(w, token, session.ExpiresAt)
	}

	slog.Info("email changed", "user_id", user.ID)
	h.writeSession(w, r, session, token)
}

// SendMagicLink sends a sign-in link and creates the account when the
// address is new. It always answers 202 to prevent email enumeration.
func (h *AuthHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	var request emailRequest
	if !Decode(w, r, &request) {
		return
	}
	email := strings.TrimSpace(request.Email)
	err := validation.ValidateEmail(email)
	if err != nil {
		ValidationError(w, map[string]string{"email": FieldMessage(err)})
		return
	}

	err = h.authService.SendMagicLink(email)
	if err != nil {
		slog.Warn("magic link send failed", "error", err)
	}
	w.WriteHeader(http.StatusAccepted)
}

// VerifyMagicLink consumes a magic link token and returns a new session.
func (h *AuthHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
	if !Decode(w, r, &request) {
		return
	}

	user, err := h.authService.VerifyMagicLink(request.Token)
	if err != nil {
		slog.Warn("magic link verification failed", "error", err)
		invalidToken(w)
		return
	}

	session, token, err := h.authService.CreateSession(r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	slog.Info("user logged in via magic link", "user_id", user.ID)
	h.writeSession(w, r, session, token)
}

// PasswordAuth signs in with email and password and returns a new session.
// Every failed attempt answers the same 401 invalid_credentials.
func (h *AuthHandler) PasswordAuth(w http.ResponseWriter, r *http.Request) {
	var request passwordLoginRequest
	if !Decode(w, r, &request) {
		return
	}
	email := strings.TrimSpace(request.Email)
	fields := map[string]string{}
	err := validation.ValidateEmail(email)
	if err != nil {
		fields["email"] = FieldMessage(err)
	}
	if request.Password == "" {
		fields["password"] = "Password is required."
	}
	if len(fields) > 0 {
		ValidationError(w, fields)
		return
	}

	user, err := h.authService.Login(email, request.Password)
	if err != nil {
		slog.Warn("password login failed", "error", err)
		Error(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password.")
		return
	}

	session, token, err := h.authService.CreateSession(r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	slog.Info("user logged in with password", "user_id", user.ID)
	h.writeSession(w, r, session, token)
}

// writeSession answers a sign-in with the raw token and the fresh user. With
// ?session=cookie it sets the HttpOnly session cookie instead and leaves the
// token out of the body. Like the web sign-in, the new cookie replaces the
// session of the cookie the browser sent, so a planted or shared session
// ends.
func (h *AuthHandler) writeSession(w http.ResponseWriter, r *http.Request, session *model.Session, token string) {
	user, err := h.userService.ByID(session.UserID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	if requestsCookieSession(r) {
		err = h.authService.EndSession(r)
		if err != nil {
			slog.Error("failed to replace the previous cookie session", "error", err, "user_id", session.UserID)
			revokeErr := h.authService.RevokeSessionByToken(token)
			if revokeErr != nil {
				slog.Error("failed to revoke the new session", "error", revokeErr, "user_id", session.UserID)
			}
			ServiceError(w, err)
			return
		}
		h.authService.SetSessionCookie(w, token, session.ExpiresAt)
		JSON(w, http.StatusOK, CookieSession{
			ExpiresAt: session.ExpiresAt.UTC(),
			User:      NewUser(user),
		})
		return
	}
	JSON(w, http.StatusOK, SessionToken{
		Token:     token,
		ExpiresAt: session.ExpiresAt.UTC(),
		User:      NewUser(user),
	})
}

// writeUser answers {"user": ...} with the fresh user.
func (h *AuthHandler) writeUser(w http.ResponseWriter, userID string) {
	user, err := h.userService.ByID(userID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, UserResponse{User: NewUser(user)})
}

// requestsCookieSession reports whether a sign-in asks for the session as
// the HttpOnly cookie of a same-origin browser app. CSRFProtection requires
// the X-CSRF-Token header on these requests.
func requestsCookieSession(r *http.Request) bool {
	return r.URL.Query().Get("session") == "cookie"
}

// isCookieCaller reports whether the request authenticates with the session
// cookie instead of an Authorization: Bearer header.
func isCookieCaller(r *http.Request) bool {
	scheme, _, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if strings.EqualFold(scheme, "Bearer") {
		return false
	}
	cookie, err := r.Cookie("session_token")
	return err == nil && cookie.Value != ""
}

func invalidToken(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "invalid_token", "This link is invalid or has expired.")
}
