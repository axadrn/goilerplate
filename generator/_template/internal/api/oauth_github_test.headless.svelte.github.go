package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeGitHubAPI answers GitHub API requests with the JSON body for each URL.
type fakeGitHubAPI map[string]string

func (f fakeGitHubAPI) RoundTrip(request *http.Request) (*http.Response, error) {
	body, ok := f[request.URL.String()]
	status := http.StatusOK
	if !ok {
		status = http.StatusNotFound
		body = `{"message":"Not Found"}`
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}, Request: request}, nil
}

func TestGitHubEmailRequiresAVerifiedAddress(t *testing.T) {
	for _, test := range []struct {
		name    string
		user    string
		emails  string
		want    string
		wantErr error
	}{
		{name: "verified public email", user: `{"email":"Public@Example.com"}`, emails: `[{"email":"primary@example.com","primary":true,"verified":true},{"email":"public@example.com","verified":true}]`, want: "public@example.com"},
		{name: "verified primary when private", user: `{"email":null}`, emails: `[{"email":"primary@example.com","primary":true,"verified":true}]`, want: "primary@example.com"},
		{name: "unverified public falls back to verified primary", user: `{"email":"victim@example.com"}`, emails: `[{"email":"victim@example.com","verified":false},{"email":"primary@example.com","primary":true,"verified":true}]`, want: "primary@example.com"},
		{name: "unverified primary", user: `{"email":null}`, emails: `[{"email":"victim@example.com","primary":true,"verified":false}]`, wantErr: errOAuthEmailUnverified},
		{name: "unverified public only", user: `{"email":"victim@example.com"}`, emails: `[{"email":"victim@example.com","primary":true,"verified":false}]`, wantErr: errOAuthEmailUnverified},
		{name: "no email", user: `{"email":null}`, emails: `[]`, wantErr: errOAuthEmailMissing},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: fakeGitHubAPI{
				"https://api.github.com/user":        test.user,
				"https://api.github.com/user/emails": test.emails,
			}}

			email, err := githubEmail(client)

			if !errors.Is(err, test.wantErr) || email != test.want {
				t.Fatalf("githubEmail() = %q, %v, want %q, %v", email, err, test.want, test.wantErr)
			}
		})
	}
}

func TestGitHubEmailFailsWhenTheEmailListIsUnavailable(t *testing.T) {
	client := &http.Client{Transport: fakeGitHubAPI{"https://api.github.com/user": `{"email":"public@example.com"}`}}

	email, err := githubEmail(client)

	if err == nil || email != "" {
		t.Fatalf("githubEmail() = %q, %v, want an error", email, err)
	}
}
