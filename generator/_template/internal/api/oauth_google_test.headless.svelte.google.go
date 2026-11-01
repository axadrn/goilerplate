package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

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
