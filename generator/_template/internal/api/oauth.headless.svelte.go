package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
)

const oauthStateCookie = "oauth_state"

var (
	errOAuthEmailMissing    = errors.New("oauth provider returned no email")
	errOAuthEmailUnverified = errors.New("oauth provider has not verified the email")
)

// OAuthHandler runs the Google and GitHub browser redirect flows. They end in
// a cookie session and send the browser to APP_URL/app. A failed sign-in goes
// to APP_URL/auth?error=oauth_failed.
type OAuthHandler struct {
	authService *service.AuthService
	cfg         *config.Config
}

func NewOAuthHandler(authService *service.AuthService, cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{authService: authService, cfg: cfg}
}

// start sends a guest to the provider with a one-time state cookie. A
// signed-in browser goes to the app instead.
func (h *OAuthHandler) start(w http.ResponseWriter, r *http.Request, authCodeURL func(state string) string) {
	if ctxkeys.User(r.Context()) != nil {
		http.Redirect(w, r, h.cfg.AppURL+"/app", http.StatusSeeOther)
		return
	}
	state, err := generateOAuthState()
	if err != nil {
		slog.Error("generate oauth state failed", "error", err)
		h.fail(w, r)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, authCodeURL(state), http.StatusTemporaryRedirect)
}

// oauthExchange trades an authorization code for an HTTP client that calls
// the provider API as the user.
type oauthExchange func(ctx context.Context, code string) (*http.Client, error)

// finish checks the state, exchanges the code, reads the account email with
// fetchEmail, and signs the browser in with a session cookie.
func (h *OAuthHandler) finish(w http.ResponseWriter, r *http.Request, provider string, exchange oauthExchange, fetchEmail func(*http.Client) (string, error)) {
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || state == "" || cookie.Value != state {
		slog.Warn("oauth state validation failed", "provider", provider, "error", err)
		h.fail(w, r)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", Path: "/", MaxAge: -1})

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("oauth callback missing code", "provider", provider)
		h.fail(w, r)
		return
	}
	client, err := exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth token exchange failed", "provider", provider, "error", err)
		h.fail(w, r)
		return
	}
	email, err := fetchEmail(client)
	if err != nil {
		slog.Error("oauth user info failed", "provider", provider, "error", err)
		h.fail(w, r)
		return
	}
	user, err := h.authService.AuthenticateOAuth(email, provider)
	if err != nil {
		slog.Error("oauth authentication failed", "provider", provider, "error", err)
		h.fail(w, r)
		return
	}
	_, err = h.authService.StartSession(w, r, user.ID)
	if err != nil {
		slog.Error("failed to create session", "provider", provider, "error", err, "user_id", user.ID)
		h.fail(w, r)
		return
	}
	slog.Info("user logged in with oauth", "provider", provider, "user_id", user.ID)
	http.Redirect(w, r, h.cfg.AppURL+"/app", http.StatusSeeOther)
}

func (h *OAuthHandler) fail(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.cfg.AppURL+"/auth?error=oauth_failed", http.StatusSeeOther)
}

func generateOAuthState() (string, error) {
	value := make([]byte, 32)
	_, err := rand.Read(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
