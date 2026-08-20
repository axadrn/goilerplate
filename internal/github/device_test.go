package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestDeviceClientCompletesAuthorization(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/device":
			if request.Form.Get("client_id") != "client-123" || request.Form.Get("scope") != "user:email" {
				t.Fatalf("device form = %v", request.Form)
			}
			json.NewEncoder(response).Encode(DeviceAuthorization{
				DeviceCode:      "device-secret",
				UserCode:        "ABCD-EFGH",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900,
				Interval:        5,
			})
		case "/token":
			polls++
			if request.Form.Get("grant_type") != deviceGrantType || request.Form.Get("device_code") != "device-secret" {
				t.Fatalf("token form = %v", request.Form)
			}
			switch polls {
			case 1:
				json.NewEncoder(response).Encode(tokenResponse{Error: "authorization_pending"})
			case 2:
				json.NewEncoder(response).Encode(tokenResponse{Error: "slow_down"})
			default:
				json.NewEncoder(response).Encode(tokenResponse{AccessToken: "github-token"})
			}
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)

	client := NewDeviceClient(server.Client())
	client.deviceURL = server.URL + "/device"
	client.tokenURL = server.URL + "/token"
	client.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	var waits []time.Duration
	client.wait = func(_ context.Context, duration time.Duration) error {
		waits = append(waits, duration)
		return nil
	}

	authorization, err := client.Start(context.Background(), "client-123")
	if err != nil {
		t.Fatal(err)
	}
	token, err := client.Wait(context.Background(), "client-123", authorization)
	if err != nil {
		t.Fatal(err)
	}
	if token != "github-token" {
		t.Fatalf("token = %q", token)
	}
	if !slices.Equal(waits, []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second}) {
		t.Fatalf("waits = %v", waits)
	}
}

func TestDeviceClientRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Write(make([]byte, (1<<20)+1))
	}))
	t.Cleanup(server.Close)
	client := NewDeviceClient(server.Client())
	client.deviceURL = server.URL
	if _, err := client.Start(context.Background(), "client-123"); err == nil {
		t.Fatal("oversized response was accepted")
	}
}
