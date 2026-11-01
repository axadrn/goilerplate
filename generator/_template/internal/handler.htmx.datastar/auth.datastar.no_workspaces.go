package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
	"{{.ModulePath}}/internal/validation"
)

type authHandler struct {
	authService         *service.AuthService
	userService         *service.UserService
	subscriptionService *service.SubscriptionService
	cfg                 *config.Config
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService, subscriptionService *service.SubscriptionService, cfg *config.Config) *authHandler {
	return &authHandler{
		authService:         authService,
		userService:         userService,
		subscriptionService: subscriptionService,
		cfg:                 cfg,
	}
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.authService.EndSession(r); err != nil {
		slog.Error("failed to revoke session during logout", "error", err)
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Sign out failed", Description: "Please try again.", Type: toast.TypeError}), "#toast-container")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Sign out failed", "Please try again."))
		}
		return
	}
	h.authService.ClearSessionCookie(w)
	redirect(w, r, "/", http.StatusSeeOther)
}

func (h *authHandler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.AuthForgotPassword(""))
}

func (h *authHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))

	if email == "" {
		render(w, r, pages.AuthForgotPasswordForm("Email is required"))
		return
	}

	err := validation.ValidateEmail(email)
	if err != nil {
		render(w, r, pages.AuthForgotPasswordForm("Please provide a valid email address"))
		return
	}

	err = h.authService.SendForgotPasswordLink(email)
	if err != nil {
		// Don't reveal specific errors to user
		slog.Warn("forgot password link send failed", "error", err)
	}

	// Always show success message to prevent email enumeration
	setAuthEmailCookie(w, r, email)
	redirect(w, r, "/auth/forgot-password-sent", http.StatusSeeOther)
}

func (h *authHandler) VerifyForgotPassword(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	user, err := h.authService.VerifyMagicLink(token)
	if err != nil {
		slog.Warn("forgot password verification failed", "error", err)
		render(w, r, pages.Auth("Invalid or expired link. Please try again."))
		return
	}

	err = h.authService.RevokeAllAccess(user.ID)
	if err != nil {
		slog.Error("failed to revoke access during forgot password flow", "error", err, "user_id", user.ID)
		render(w, r, pages.Auth("An error occurred. Please try again."))
		return
	}

	if user.HasPassword() {
		err = h.authService.RemovePassword(user.ID)
		if err != nil {
			slog.Error("failed to remove password during forgot password flow", "error", err, "user_id", user.ID)
			render(w, r, pages.Auth("An error occurred. Please try again."))
			return
		}
		slog.Info("password removed via forgot password flow", "user_id", user.ID)
	}

	_, err = h.authService.StartFreshSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		render(w, r, pages.Auth("An error occurred. Please try again."))
		return
	}

	slog.Info("user logged in via forgot password flow", "user_id", user.ID)

	// Redirect to settings with query param for toast notification
	redirect(w, r, "/app/settings?tab=security&password_removed=1", http.StatusSeeOther)
}

func (h *authHandler) VerifyEmailChange(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	user, err := h.authService.VerifyEmailChange(token)
	if err != nil {
		slog.Warn("email change verification failed", "error", err)
		render(w, r, pages.VerifyEmailError("Invalid or expired verification link"))
		return
	}

	_, err = h.authService.StartFreshSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session after email change", "error", err, "user_id", user.ID)
		render(w, r, pages.VerifyEmailError("An error occurred. Please try again."))
		return
	}

	slog.Info("email changed", "user_id", user.ID)
	render(w, r, pages.VerifyEmailSuccess())
}

func (h *authHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))

	if email == "" {
		render(w, r, pages.AuthMagicLinkForm("Email is required"))
		return
	}

	err := validation.ValidateEmail(email)
	if err != nil {
		render(w, r, pages.AuthMagicLinkForm("Please provide a valid email address"))
		return
	}

	err = h.authService.SendMagicLink(email)
	if err != nil {
		// Don't reveal specific errors to prevent email enumeration
		slog.Warn("magic link send failed", "error", err)
	}

	if r.URL.Query().Get("resend") == "true" {
		render(w, r, pages.AuthMagicLinkResend(email, true))
		return
	}

	setAuthEmailCookie(w, r, email)
	redirect(w, r, "/auth/magic-link-sent", http.StatusSeeOther)
}

func (h *authHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	user, err := h.authService.VerifyMagicLink(token)
	if err != nil {
		slog.Warn("magic link verification failed", "error", err)
		render(w, r, pages.Auth("Invalid or expired magic link. Please try again."))
		return
	}

	_, err = h.authService.StartSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		render(w, r, pages.Auth("An error occurred. Please try again."))
		return
	}

	needsOnboarding, err := h.authService.NeedsOnboarding(user.ID)
	if err != nil {
		slog.Warn("failed to check onboarding status", "error", err, "user_id", user.ID)
	}

	if needsOnboarding {
		slog.Info("new user needs onboarding", "user_id", user.ID)
		redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
		return
	}

	slog.Info("user logged in via magic link", "user_id", user.ID)
	redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (h *authHandler) AuthPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.Auth(""))
}

func (h *authHandler) PasswordPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.AuthPassword(""))
}

func (h *authHandler) OnboardingPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.AuthOnboarding(""))
}

func (h *authHandler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	err := h.authService.CompleteOnboarding(user.ID, name)
	if err != nil {
		slog.Error("onboarding failed", "error", err, "user_id", user.ID)
		render(w, r, pages.AuthOnboardingForm("Please enter your name"))
		return
	}

	slog.Info("onboarding completed", "user_id", user.ID)
	redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (h *authHandler) PasswordAuth(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		render(w, r, pages.AuthPasswordForm("Email and password are required"))
		return
	}

	err := validation.ValidateEmail(email)
	if err != nil {
		render(w, r, pages.AuthPasswordForm("Please provide a valid email address"))
		return
	}

	user, err := h.authService.Login(email, password)
	if err != nil {
		slog.Warn("password login failed", "error", err)
		render(w, r, pages.AuthPasswordForm("Invalid email or password"))
		return
	}

	_, err = h.authService.StartSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		render(w, r, pages.AuthPasswordForm("An error occurred. Please try again."))
		return
	}

	slog.Info("user logged in with password", "user_id", user.ID)
	redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (h *authHandler) redirectAfterAuth(w http.ResponseWriter, r *http.Request, user *model.User) {
	needsOnboarding, err := h.authService.NeedsOnboarding(user.ID)
	if err != nil {
		slog.Warn("failed to check onboarding status", "error", err, "user_id", user.ID)
	}
	if needsOnboarding {
		redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
		return
	}
	redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}
