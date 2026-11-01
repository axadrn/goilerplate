package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func (h *OAuthHandler) googleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GoogleClientID,
		ClientSecret: h.cfg.GoogleClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

// GoogleAuth redirects the browser to the Google consent screen.
func (h *OAuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	h.start(w, r, func(state string) string {
		return h.googleConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	})
}

// GoogleCallback finishes the Google sign-in.
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthConfig := h.googleConfig()
	exchange := func(ctx context.Context, code string) (*http.Client, error) {
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		return oauthConfig.Client(ctx, token), nil
	}
	h.finish(w, r, "google", exchange, googleEmail)
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

