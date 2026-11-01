package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/api"
)

func TestListSessionsMarksCurrent(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "lister", "Lister")
	token := environment.signIn(t, user.ID)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Version/17.0 Mobile/15E148 Safari/604.1")
	_, _, err := environment.auth.CreateSession(request, user.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account/sessions", bearer: token})
	expectStatus(t, recorder, http.StatusOK)
	for _, hidden := range []string{"token_hash", "user_agent", "Mozilla"} {
		if strings.Contains(recorder.Body.String(), hidden) {
			t.Fatalf("session list exposes %s: %s", hidden, recorder.Body.String())
		}
	}
	list := decodeBody[api.SessionList](t, recorder)
	if len(list.Items) != 2 {
		t.Fatalf("items = %#v", list.Items)
	}
	current := 0
	phone := false
	for _, session := range list.Items {
		if session.Current {
			current++
		}
		if session.DisplayName == "Safari on iPhone" {
			phone = true
		}
		if session.ID == "" || session.CreatedAt.IsZero() || session.LastSeenAt.IsZero() {
			t.Fatalf("session = %#v", session)
		}
	}
	if current != 1 || !phone {
		t.Fatalf("current = %d, phone = %t, items %#v", current, phone, list.Items)
	}
}

func sessionIDs(t *testing.T, environment *apiEnvironment, token string) (string, []string) {
	t.Helper()
	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account/sessions", bearer: token})
	expectStatus(t, recorder, http.StatusOK)
	currentID := ""
	var others []string
	for _, session := range decodeBody[api.SessionList](t, recorder).Items {
		if session.Current {
			currentID = session.ID
			continue
		}
		others = append(others, session.ID)
	}
	return currentID, others
}

func TestRevokeOtherSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "revoker", "Revoker")
	token := environment.signIn(t, user.ID)
	other := environment.signIn(t, user.ID)
	_, others := sessionIDs(t, environment, token)

	recorder := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions/" + others[0], bearer: token})
	expectStatus(t, recorder, http.StatusNoContent)
	expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: other}), http.StatusUnauthorized, "unauthenticated")
	expectStatus(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: token}), http.StatusOK)

	missing := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions/" + others[0], bearer: token})
	expectError(t, missing, http.StatusNotFound, "not_found")
}

func TestRevokeSessionOfAnotherUserIsNotFound(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "owner", "Owner")
	stranger := seedVerifiedUser(t, environment, "stranger", "Stranger")
	token := environment.signIn(t, user.ID)
	strangerToken := environment.signIn(t, stranger.ID)
	strangerID, _ := sessionIDs(t, environment, strangerToken)

	recorder := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions/" + strangerID, bearer: token})
	expectError(t, recorder, http.StatusNotFound, "not_found")
	expectStatus(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: strangerToken}), http.StatusOK)
}

func TestRevokeCurrentSessionEndsIt(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "self", "Self")
	token := environment.signIn(t, user.ID)
	currentID, _ := sessionIDs(t, environment, token)

	recorder := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions/" + currentID, bearer: token})
	expectStatus(t, recorder, http.StatusNoContent)
	expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: token}), http.StatusUnauthorized, "unauthenticated")
}

func TestRevokeCurrentCookieSessionClearsCookie(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "cookie", "Cookie")
	token := environment.signIn(t, user.ID)
	currentID, _ := sessionIDs(t, environment, token)

	recorder := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions/" + currentID, cookie: token})
	expectStatus(t, recorder, http.StatusNoContent)
	cleared := false
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("session cookie was not cleared: %#v", recorder.Result().Cookies())
	}
}

func TestRevokeAllOtherSessions(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "everywhere", "Everywhere")
	token := environment.signIn(t, user.ID)
	first := environment.signIn(t, user.ID)
	second := environment.signIn(t, user.ID)

	recorder := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions", bearer: token})
	expectStatus(t, recorder, http.StatusNoContent)
	for _, revoked := range []string{first, second} {
		expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: revoked}), http.StatusUnauthorized, "unauthenticated")
	}
	currentID, others := sessionIDs(t, environment, token)
	if currentID == "" || len(others) != 0 {
		t.Fatalf("current = %q, others = %#v", currentID, others)
	}
}
