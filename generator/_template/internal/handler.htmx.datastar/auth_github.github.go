package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/ui/pages"
)

func (h *authHandler) githubOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GitHubClientID,
		ClientSecret: h.cfg.GitHubClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

// GitHubAuth redirects user to GitHub OAuth consent screen
func (h *authHandler) GitHubAuth(w http.ResponseWriter, r *http.Request) {
	// Generate secure state token for CSRF protection
	state := generateOAuthState()

	// Get config from context to determine if we're in production
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

	url := h.githubOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubCallback handles the OAuth callback from GitHub
func (h *authHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter for CSRF protection
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state || state == "" {
		slog.Warn("github oauth state validation failed", "error", err)
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
		slog.Warn("github oauth callback missing code")
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	// Exchange code for token
	token, err := h.githubOAuthConfig().Exchange(context.Background(), code)
	if err != nil {
		slog.Error("github oauth token exchange failed", "error", err)
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	client := h.githubOAuthConfig().Client(context.Background(), token)
	email, err := githubEmail(client)
	if errors.Is(err, errOAuthEmailMissing) || errors.Is(err, errOAuthEmailUnverified) {
		slog.Warn("github oauth: no verified email", "error", err)
		render(w, r, pages.Auth("GitHub did not share a verified email address. Verify your email on GitHub and try again."))
		return
	}
	if err != nil {
		slog.Error("failed to get github user info", "error", err)
		render(w, r, pages.Auth("OAuth authentication failed. Please try again."))
		return
	}

	// Authenticate or create user
	user, err := h.authService.AuthenticateOAuth(email, "github")
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

	slog.Info("user logged in with github oauth", "user_id", user.ID)

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

// githubEmail returns the public email, or the primary address when the
// public one is private. Either must be verified on GitHub, so nobody signs
// in to an account by adding its address to GitHub without proving it.
func githubEmail(client *http.Client) (string, error) {
	var userInfo struct {
		Email string `json:"email"`
	}
	err := getOAuthJSON(client, "https://api.github.com/user", &userInfo)
	if err != nil {
		return "", err
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	err = getOAuthJSON(client, "https://api.github.com/user/emails", &emails)
	if err != nil {
		return "", err
	}
	primary := ""
	for _, email := range emails {
		if !email.Verified || email.Email == "" {
			continue
		}
		if userInfo.Email != "" && strings.EqualFold(email.Email, userInfo.Email) {
			return email.Email, nil
		}
		if email.Primary {
			primary = email.Email
		}
	}
	if primary != "" {
		return primary, nil
	}
	if len(emails) == 0 && userInfo.Email == "" {
		return "", errOAuthEmailMissing
	}
	return "", errOAuthEmailUnverified
}

func getOAuthJSON(client *http.Client, url string, value any) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := response.Body.Close()
		if closeErr != nil {
			slog.Error("failed to close response body", "error", closeErr)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d", url, response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(value)
}
