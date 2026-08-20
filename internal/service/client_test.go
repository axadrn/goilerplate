package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/axadrn/goilerplate/api"
)

func TestClientLoginAndWhoAmI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case api.PathGitHubLogin:
			var input api.GitHubLoginRequest
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			if input.OAuthAccessToken != "github-token" {
				t.Fatalf("OAuth token = %q", input.OAuthAccessToken)
			}
			json.NewEncoder(response).Encode(api.GitHubLoginResponse{
				SessionToken: "session-token",
				Account:      api.Account{ID: "user-1", GitHubLogin: "axadrn", Email: "hello@example.com"},
			})
		case api.PathWhoAmI:
			if request.Header.Get("Authorization") != "Bearer session-token" {
				t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
			}
			json.NewEncoder(response).Encode(api.WhoAmIResponse{
				Account:            api.Account{ID: "user-1", GitHubLogin: "axadrn", Email: "hello@example.com"},
				Licenses:           []api.License{{ID: "license-1", Tier: api.LicenseTierPaid, Status: api.LicenseStatusActive, Role: api.LicenseRoleOwner}},
				EffectiveLicenseID: "license-1",
			})
		case api.PathLogout:
			if request.Header.Get("Authorization") != "Bearer session-token" {
				t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
			}
			response.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}

	login, err := client.LoginWithGitHub(context.Background(), "github-token")
	if err != nil {
		t.Fatal(err)
	}
	who, err := client.WhoAmI(context.Background(), login.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	if who.Account.GitHubLogin != "axadrn" || who.EffectiveLicenseID != "license-1" {
		t.Fatalf("whoami = %#v", who)
	}
	if err := client.Logout(context.Background(), login.SessionToken); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsInsecureRemoteURL(t *testing.T) {
	if _, err := NewClient("http://example.com", nil); err == nil {
		t.Fatal("insecure remote API URL was accepted")
	}
}

func TestClientBoundsAndReportsErrors(t *testing.T) {
	t.Run("API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(response).Encode(api.ErrorResponse{Code: "unauthorized", Message: "sign in again"})
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(server.URL, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.WhoAmI(context.Background(), "bad-token"); err == nil || !strings.Contains(err.Error(), "sign in again") {
			t.Fatalf("WhoAmI() error = %v", err)
		}
	})

	t.Run("oversized response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Write(make([]byte, maxResponseSize+1))
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(server.URL, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.WhoAmI(context.Background(), "token"); err == nil {
			t.Fatal("oversized response was accepted")
		}
	})
}
