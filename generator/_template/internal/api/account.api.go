package api

import (
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/validation"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type setPasswordRequest struct {
	Password string `json:"password"`
}

type deleteAccountRequest struct {
	ConfirmEmail string `json:"confirm_email"`
}

// UpdateName changes the display name and returns the user.
func (h *AccountHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request nameRequest
	if !Decode(w, r, &request) {
		return
	}
	err := validation.ValidateName(request.Name)
	if err != nil {
		ValidationError(w, map[string]string{"name": FieldMessage(err)})
		return
	}

	updatedUser, err := h.userService.UpdateName(user.ID, request.Name)
	if err != nil {
		slog.Error("failed to update name", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, UserResponse{User: NewUser(updatedUser)})
}

// ChangeEmail sends a verification link to the new address. The address
// changes once the link is verified.
func (h *AccountHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request emailRequest
	if !Decode(w, r, &request) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(request.Email))
	err := validation.ValidateEmail(email)
	if err != nil {
		ValidationError(w, map[string]string{"email": FieldMessage(err)})
		return
	}
	if email == user.Email {
		ValidationError(w, map[string]string{"email": "Email address is already set to this value."})
		return
	}

	err = h.authService.RequestEmailChange(user.ID, email)
	if err != nil {
		slog.Warn("email change request failed", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	slog.Info("email change requested", "user_id", user.ID)
	w.WriteHeader(http.StatusAccepted)
}

// ChangePassword replaces the password, signs out every other session and
// revokes every personal API token.
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request changePasswordRequest
	if !Decode(w, r, &request) {
		return
	}
	fields := map[string]string{}
	if request.CurrentPassword == "" {
		fields["current_password"] = "Current password is required."
	}
	err := validation.ValidatePassword(request.NewPassword)
	if err != nil {
		fields["new_password"] = FieldMessage(err)
	}
	if len(fields) > 0 {
		ValidationError(w, fields)
		return
	}
	if !user.HasPassword() {
		Error(w, http.StatusConflict, "password_not_set", "This account has no password. Set one first.")
		return
	}

	err = h.userService.UpdatePassword(user.ID, request.CurrentPassword, request.NewPassword)
	if err != nil {
		slog.Warn("password update failed", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	h.revokeOtherAccess(r, user.ID)
	slog.Info("password updated", "user_id", user.ID)
	NoContent(w)
}

// SetPassword adds a password to a passwordless account, signs out every
// other session and revokes every personal API token.
func (h *AccountHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request setPasswordRequest
	if !Decode(w, r, &request) {
		return
	}
	err := validation.ValidatePassword(request.Password)
	if err != nil {
		ValidationError(w, map[string]string{"password": FieldMessage(err)})
		return
	}
	if user.HasPassword() {
		Error(w, http.StatusConflict, "password_already_set", "This account already has a password. Change it instead.")
		return
	}

	err = h.authService.SetPassword(user.ID, request.Password)
	if err != nil {
		slog.Warn("set password failed", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	h.revokeOtherAccess(r, user.ID)
	slog.Info("password set", "user_id", user.ID)
	NoContent(w)
}

// RemovePassword makes the account passwordless, signs out every other
// session and revokes every personal API token.
func (h *AccountHandler) RemovePassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if !user.HasPassword() {
		Error(w, http.StatusConflict, "password_not_set", "This account has no password.")
		return
	}

	err := h.authService.RemovePassword(user.ID)
	if err != nil {
		slog.Warn("remove password failed", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	h.revokeOtherAccess(r, user.ID)
	slog.Info("password removed", "user_id", user.ID)
	NoContent(w)
}

// revokeOtherAccess signs out every session except the current one and
// revokes every personal API token. A failure is logged and does not undo the
// password change.
func (h *AccountHandler) revokeOtherAccess(r *http.Request, userID string) {
	session := ctxkeys.Session(r.Context())
	if session == nil {
		return
	}
	err := h.authService.RevokeOtherAccess(userID, session.ID)
	if err != nil {
		slog.Error("failed to revoke other sessions and api tokens after password update", "error", err, "user_id", userID)
	}
}

// DeleteAccount deletes the account after the caller confirms its email
// address. Every session of the account ends with it.
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request deleteAccountRequest
	if !Decode(w, r, &request) {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(request.ConfirmEmail), user.Email) {
		ValidationError(w, map[string]string{"confirm_email": "Type your email address to confirm."})
		return
	}

	err := h.userService.DeleteAccount(user.ID)
	if err != nil {
		accountDeletionError(w, user.ID, err)
		return
	}

	slog.Info("account deleted", "user_id", user.ID)
	if !ctxkeys.IsBearer(r.Context()) {
		h.authService.ClearSessionCookie(w)
	}
	NoContent(w)
}
