package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
	"{{.ModulePath}}/internal/validation"
)

func (h *AccountHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	email := r.FormValue("email")
	email = strings.TrimSpace(strings.ToLower(email))

	err := validation.ValidateEmail(email)
	if err != nil {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "Valid email is required",
			Type:        toast.TypeError,
		}))
		return
	}

	if email == user.Email {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Info",
			Description: "Email address is already set to this value",
			Type:        toast.TypeInfo,
		}))
		return
	}

	err = h.authService.RequestEmailChange(user.ID, email)
	if err != nil {
		slog.Warn("email change request failed", "error", err, "user_id", user.ID)

		errMsg := "Failed to change email"
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			errMsg = "Email already in use"
		} else if errors.Is(err, service.ErrInvalidEmail) {
			errMsg = "Invalid email address"
		}

		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: errMsg,
			Type:        toast.TypeError,
		}))
		return
	}

	slog.Info("email change requested", "user_id", user.ID)
	accountToast(w, r, toast.Toast(toast.Props{
		Title:       "Verification Required",
		Description: "Please check your new email address to verify the change",
		Type:        toast.TypeInfo,
	}))
}

func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "All password fields are required",
			Type:        toast.TypeError,
		}))
		return
	}

	if newPassword != confirmPassword {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "New passwords do not match",
			Type:        toast.TypeError,
		}))
		return
	}

	err := validation.ValidatePassword(newPassword)
	if err != nil {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: err.Error(),
			Type:        toast.TypeError,
		}))
		return
	}

	err = h.userService.UpdatePassword(user.ID, currentPassword, newPassword)
	if err != nil {
		slog.Warn("password update failed", "error", err, "user_id", user.ID)

		errMsg := "Failed to update password"
		if errors.Is(err, service.ErrInvalidCurrentPassword) {
			errMsg = "Current password is incorrect"
		}

		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: errMsg,
			Type:        toast.TypeError,
		}))
		return
	}

	revocationErr := h.revokeOtherAccess(r, user.ID)
	if revocationErr != nil {
		slog.Error("failed to revoke other sessions after password update", "error", revocationErr, "user_id", user.ID)
	}

	slog.Info("password updated", "user_id", user.ID)
	toastProps := toast.Props{
		Title:       "Success",
		Description: "Password updated successfully",
		Type:        toast.TypeSuccess,
	}
	if revocationErr != nil {
		toastProps = toast.Props{
			Title:       "Password Updated",
			Description: "Your password changed, but other sessions could not be signed out. Use Sign Out Everywhere and try again.",
			Type:        toast.TypeWarning,
			Timeout:     8000,
		}
	}
	accountToast(w, r, toast.Toast(toastProps))
	renderFragment(w, r, pages.SettingsPasswordSection(), "settings-password-form")
}

func (h *AccountHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if newPassword == "" || confirmPassword == "" {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "All fields are required",
			Type:        toast.TypeError,
		}))
		return
	}

	if newPassword != confirmPassword {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "Passwords do not match",
			Type:        toast.TypeError,
		}))
		return
	}

	err := validation.ValidatePassword(newPassword)
	if err != nil {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: err.Error(),
			Type:        toast.TypeError,
		}))
		return
	}

	err = h.authService.SetPassword(user.ID, newPassword)
	if err != nil {
		slog.Warn("set password failed", "error", err, "user_id", user.ID)

		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Failed to Set Password",
			Description: err.Error(),
			Type:        toast.TypeError,
			Timeout:     8000, // 8 seconds for errors
		}))
		return
	}

	// Reload user from DB to get updated password_hash
	updatedUser, err := h.userService.ByID(user.ID)
	if err != nil {
		slog.Error("failed to reload user after password set", "error", err, "user_id", user.ID)
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "Password set but failed to refresh. Please reload the page.",
			Type:        toast.TypeError,
		}))
		return
	}

	// Update context with fresh user
	ctx := ctxkeys.WithUser(r.Context(), updatedUser)
	revocationErr := h.revokeOtherAccess(r, user.ID)
	if revocationErr != nil {
		slog.Error("failed to revoke other sessions after password set", "error", revocationErr, "user_id", user.ID)
	}

	slog.Info("password set", "user_id", user.ID)
	toastProps := toast.Props{
		Title:       "Success",
		Description: "Password set successfully. You can now sign in with your password.",
		Type:        toast.TypeSuccess,
	}
	if revocationErr != nil {
		toastProps = toast.Props{
			Title:       "Password Set",
			Description: "Your password was set, but other sessions could not be signed out. Use Sign Out Everywhere and try again.",
			Type:        toast.TypeWarning,
			Timeout:     8000,
		}
	}
	accountToast(w, r, toast.Toast(toastProps))
	renderFragment(w, r.WithContext(ctx), pages.SettingsPasswordSection(), "settings-password-form")
}

func (h *AccountHandler) RemovePassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	err := h.authService.RemovePassword(user.ID)
	if err != nil {
		slog.Warn("remove password failed", "error", err, "user_id", user.ID)

		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Failed to Remove Password",
			Description: err.Error(),
			Type:        toast.TypeError,
			Timeout:     8000, // 8 seconds for errors
		}))
		return
	}

	// Reload user from DB to get updated password_hash (now NULL)
	updatedUser, err := h.userService.ByID(user.ID)
	if err != nil {
		slog.Error("failed to reload user after password removal", "error", err, "user_id", user.ID)
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "Password removed but failed to refresh. Please reload the page.",
			Type:        toast.TypeError,
		}))
		return
	}

	// Update context with fresh user
	ctx := ctxkeys.WithUser(r.Context(), updatedUser)
	revocationErr := h.revokeOtherAccess(r, user.ID)
	if revocationErr != nil {
		slog.Error("failed to revoke other sessions after password removal", "error", revocationErr, "user_id", user.ID)
	}

	slog.Info("password removed", "user_id", user.ID)

	toastProps := toast.Props{
		Title:       "Success",
		Description: "Password removed. You can now only sign in with magic links.",
		Type:        toast.TypeSuccess,
	}
	if revocationErr != nil {
		toastProps = toast.Props{
			Title:       "Password Removed",
			Description: "Your password was removed, but other sessions could not be signed out. Use Sign Out Everywhere and try again.",
			Type:        toast.TypeWarning,
			Timeout:     8000,
		}
	}
	accountToast(w, r, toast.Toast(toastProps))
	renderFragment(w, r.WithContext(ctx), pages.SettingsPasswordSection(), "settings-password-form")
}

func (h *AccountHandler) revokeOtherAccess(r *http.Request, userID string) error {
	session := ctxkeys.Session(r.Context())
	if session == nil {
		return nil
	}
	return h.authService.RevokeOtherAccess(userID, session.ID)
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	confirmation := strings.TrimSpace(r.FormValue("delete_confirmation"))

	if confirmation != "DELETE" {
		accountToast(w, r, toast.Toast(toast.Props{
			Title:       "Error",
			Description: "Please type DELETE to confirm account deletion",
			Type:        toast.TypeError,
		}))
		return
	}

	err := h.userService.DeleteAccount(user.ID)
	if err != nil {
		accountDeletionError(w, r, user.ID, err)
		return
	}

	slog.Info("account deleted", "user_id", user.ID)
	h.authService.ClearSessionCookie(w)

	redirect(w, r, "/", http.StatusSeeOther)
}
