// Package githubstars asks GitHub how many stars the goilerplate repo has.
// The site build calls it once, without a token, and bakes the number into
// the pages.
package githubstars

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Repo is the repo whose stars the navbar shows.
const Repo = "axadrn/goilerplate"

// APIURL is where the stars come from; tests point it at a fake.
var APIURL = "https://api.github.com"

// Fetch returns the stars GitHub reports for Repo.
func Fetch(ctx context.Context, client *http.Client) (int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, APIURL+"/repos/"+Repo, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("github: %s", response.Status)
	}
	var repo struct {
		Stars int `json:"stargazers_count"`
	}
	err = json.NewDecoder(response.Body).Decode(&repo)
	return repo.Stars, err
}

// Short writes a count the way GitHub does: 950, 1.2k, 12k.
func Short(stars int) string {
	switch {
	case stars < 1000:
		return fmt.Sprint(stars)
	case stars < 10000:
		return fmt.Sprintf("%.1fk", float64(stars)/1000)
	default:
		return fmt.Sprintf("%dk", stars/1000)
	}
}
