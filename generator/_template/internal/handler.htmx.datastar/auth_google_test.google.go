package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
)

func TestGoogleAuthUsesOneTimeStateCookie(t *testing.T) {
	handler := &authHandler{cfg: &config.Config{AppURL: "https://app.example.com", APIURL: "https://api.example.com", GoogleClientID: "client"}}
	request := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	request = request.WithContext(ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{AppEnv: "production"}))
	response := httptest.NewRecorder()

	handler.GoogleAuth(response, request)
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	cookies := response.Result().Cookies()
	if response.Code != http.StatusTemporaryRedirect || len(cookies) != 1 {
		t.Fatalf("GoogleAuth() status = %d, cookies = %#v", response.Code, cookies)
	}
	cookie := cookies[0]
	if cookie.Name != "oauth_state" || cookie.Value == "" || cookie.Value != location.Query().Get("state") {
		t.Fatalf("OAuth state cookie = %#v, redirect = %q", cookie, location.String())
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 600 {
		t.Fatalf("OAuth state cookie flags = %#v", cookie)
	}
	redirectURI := location.Query().Get("redirect_uri")
	if redirectURI != "https://api.example.com/auth/google/callback" {
		t.Fatalf("redirect_uri = %q, want the API_URL callback", redirectURI)
	}
}

// fakeGoogleAPI answers the Google userinfo request with body.
type fakeGoogleAPI string

func (f fakeGoogleAPI) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.String() != "https://www.googleapis.com/oauth2/v2/userinfo" {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`)), Request: request}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(f))), Header: http.Header{"Content-Type": {"application/json"}}, Request: request}, nil
}

func TestGoogleEmailRequiresAVerifiedAddress(t *testing.T) {
	for _, test := range []struct {
		name     string
		userinfo string
		want     string
		wantErr  error
	}{
		{name: "verified", userinfo: `{"email":"person@example.com","verified_email":true}`, want: "person@example.com"},
		{name: "unverified", userinfo: `{"email":"victim@example.com","verified_email":false}`, wantErr: errOAuthEmailUnverified},
		{name: "verification unknown", userinfo: `{"email":"victim@example.com"}`, wantErr: errOAuthEmailUnverified},
		{name: "no email", userinfo: `{"verified_email":true}`, wantErr: errOAuthEmailMissing},
	} {
		t.Run(test.name, func(t *testing.T) {
			email, err := googleEmail(&http.Client{Transport: fakeGoogleAPI(test.userinfo)})

			if !errors.Is(err, test.wantErr) || email != test.want {
				t.Fatalf("googleEmail() = %q, %v, want %q, %v", email, err, test.want, test.wantErr)
			}
		})
	}
}
