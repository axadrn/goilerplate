package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
		case api.PathActivation:
			if request.Header.Get("Authorization") != "Bearer session-token" {
				t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
			}
			json.NewEncoder(response).Encode(api.ActivationStatusResponse{Active: true})
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
	activation, err := client.ActivationStatus(context.Background(), login.SessionToken)
	if err != nil || !activation.Active {
		t.Fatalf("activation = %#v, %v", activation, err)
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

func TestClientDownloadsGeneratedProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != api.PathGenerate || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer session-token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var request api.GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Answers.ModulePath != "example.com/project" {
			t.Fatalf("request = %#v", request)
		}
		w.Header().Set("X-Goilerplate-Version", "v3.0.0")
		w.Write([]byte("archive-bytes"))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	version, err := client.Generate(context.Background(), "session-token", api.GenerateRequest{
		Answers: api.GenerationAnswers{ModulePath: "example.com/project"},
	}, &destination)
	if err != nil {
		t.Fatal(err)
	}
	if version != "v3.0.0" || destination.String() != "archive-bytes" {
		t.Fatalf("version = %q, archive = %q", version, destination.String())
	}
}

func TestClientRejectsGeneratedProjectWithoutVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("archive-bytes"))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), "session-token", api.GenerateRequest{}, io.Discard); err == nil {
		t.Fatal("missing template version was accepted")
	}
}
