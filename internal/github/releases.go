package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const releasesURL = "https://api.github.com/repos/axadrn/goilerplate/releases?per_page=10"
const maxReleasesResponseSize = 1 << 20

type Release struct {
	Tag         string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
}

func ListReleases(ctx context.Context, client *http.Client) ([]Release, error) {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "goilerplate")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load release notes: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load release notes: GitHub returned %s", response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxReleasesResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read release notes: %w", err)
	}
	if len(content) > maxReleasesResponseSize {
		return nil, errors.New("release notes response is too large")
	}
	var releases []Release
	if err := json.Unmarshal(content, &releases); err != nil {
		return nil, fmt.Errorf("read release notes: %w", err)
	}
	published := releases[:0]
	for _, release := range releases {
		if !release.Draft && release.Tag != "" {
			published = append(published, release)
		}
	}
	return published, nil
}
