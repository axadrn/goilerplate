package github

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestListReleasesReturnsPublishedReleases(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != releasesURL || request.Header.Get("User-Agent") != "goilerplate" {
			t.Fatalf("request = %s, headers = %#v", request.URL, request.Header)
		}
		body := `[
			{"tag_name":"v3.0.0-beta.1","name":"Beta","body":"Ready","published_at":"2026-08-21T10:00:00Z","prerelease":true},
			{"tag_name":"v3.1.0","draft":true}
		]`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	releases, err := ListReleases(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Tag != "v3.0.0-beta.1" || releases[0].Body != "Ready" || !releases[0].Prerelease {
		t.Fatalf("releases = %#v", releases)
	}
}

func TestListReleasesExplainsRateLimitReset(t *testing.T) {
	reset := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "403 Forbidden",
			Header:     http.Header{"X-Ratelimit-Reset": []string{strconv.FormatInt(reset.Unix(), 10)}},
			Body:       io.NopCloser(strings.NewReader("rate limited")),
		}, nil
	})}
	_, err := ListReleases(context.Background(), client)
	if err == nil || !strings.Contains(err.Error(), reset.Format(time.RFC3339)) {
		t.Fatalf("ListReleases() error = %v", err)
	}
}

func TestListReleasesRejectsFailedAndOversizedResponses(t *testing.T) {
	for name, response := range map[string]*http.Response{
		"status": {StatusCode: http.StatusForbidden, Status: "403 Forbidden", Body: io.NopCloser(strings.NewReader("no"))},
		"size":   {StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxReleasesResponseSize+1)))},
	} {
		t.Run(name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return response, nil
			})}
			if _, err := ListReleases(context.Background(), client); err == nil {
				t.Fatal("ListReleases accepted invalid response")
			}
		})
	}
}
