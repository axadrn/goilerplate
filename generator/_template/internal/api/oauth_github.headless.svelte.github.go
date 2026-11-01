package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func (h *OAuthHandler) githubConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GitHubClientID,
		ClientSecret: h.cfg.GitHubClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

// GitHubAuth redirects the browser to the GitHub consent screen.
func (h *OAuthHandler) GitHubAuth(w http.ResponseWriter, r *http.Request) {
	h.start(w, r, func(state string) string {
		return h.githubConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	})
}

// GitHubCallback finishes the GitHub sign-in.
func (h *OAuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	oauthConfig := h.githubConfig()
	exchange := func(ctx context.Context, code string) (*http.Client, error) {
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		return oauthConfig.Client(ctx, token), nil
	}
	h.finish(w, r, "github", exchange, githubEmail)
}

// githubEmail returns the public email, or the primary address when the
// public one is private. Either must be verified on GitHub, so nobody signs
// in to an account by adding its address to GitHub without proving it.
func githubEmail(client *http.Client) (string, error) {
	var userInfo struct {
		Email string `json:"email"`
	}
	err := getJSON(client, "https://api.github.com/user", &userInfo)
	if err != nil {
		return "", err
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	err = getJSON(client, "https://api.github.com/user/emails", &emails)
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

func getJSON(client *http.Client, url string, value any) error {
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
