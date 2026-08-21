package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
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
			{"tag_name":"v3.0.0","name":"Launch","body":"Ready","published_at":"2026-08-21T10:00:00Z"},
			{"tag_name":"v3.1.0","draft":true}
		]`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	releases, err := ListReleases(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Tag != "v3.0.0" || releases[0].Body != "Ready" {
		t.Fatalf("releases = %#v", releases)
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
