package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/ui/pages"
)

func (h *authHandler) googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GoogleClientID,
		ClientSecret: h.cfg.GoogleClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

// GoogleAuth redirects user to Google OAuth consent screen
func (h *authHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	// Generate secure state token for CSRF protection
	state := generateOAuthState()

	cfg := ctxkeys.RenderConfig(r.Context())
	isProduction := cfg.IsProduction()

	// Store state in secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction, // Secure flag based on APP_ENV (safer than r.TLS behind load balancers)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	url := h.googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the OAuth callback from Google
func (h *authHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter for CSRF protection
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state || state == "" {
		slog.Warn("google oauth state validation failed", "error", err)
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("google oauth callback missing code")
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	// Exchange code for token
	token, err := h.googleOAuthConfig().Exchange(context.Background(), code)
	if err != nil {
		slog.Error("google oauth token exchange failed", "error", err)
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	client := h.googleOAuthConfig().Client(context.Background(), token)
	email, err := googleEmail(client)
	if errors.Is(err, errOAuthEmailMissing) || errors.Is(err, errOAuthEmailUnverified) {
		slog.Warn("google oauth: no verified email", "error", err)
		render(w, r, pages.Auth("Google did not share a verified email address. Verify your email with Google and try again."))
		return
	}
	if err != nil {
		slog.Error("failed to get google user info", "error", err)
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	// Authenticate or create user
	user, err := h.authService.AuthenticateOAuth(email, "google")
	if err != nil {
		slog.Error("oauth authentication failed", "error", err)
		render(w, r, pages.Auth("Authentication failed. Please try again."))
		return
	}

	_, err = h.authService.StartSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err, "user_id", user.ID)
		render(w, r, pages.Auth("An error occurred. Please try again."))
		return
	}

	slog.Info("user logged in with google oauth", "user_id", user.ID)

	// Check if user needs onboarding
	needsOnboarding, err := h.authService.NeedsOnboarding(user.ID)
	if err != nil {
		slog.Warn("failed to check onboarding status", "error", err, "user_id", user.ID)
	}

	if needsOnboarding {
		redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
	} else {
		h.redirectAfterAuth(w, r, user)
	}
}

// googleEmail returns the account email, which Google must have verified.
func googleEmail(client *http.Client) (string, error) {
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", err
	}
	defer func() {
		closeErr := response.Body.Close()
		if closeErr != nil {
			slog.Error("failed to close response body", "error", closeErr)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google userinfo answered %d", response.StatusCode)
	}
	var userInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	err = json.NewDecoder(response.Body).Decode(&userInfo)
	if err != nil {
		return "", err
	}
	if userInfo.Email == "" {
		return "", errOAuthEmailMissing
	}
	if !userInfo.VerifiedEmail {
		return "", errOAuthEmailUnverified
	}
	return userInfo.Email, nil
}
