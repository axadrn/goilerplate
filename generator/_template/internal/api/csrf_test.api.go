package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type csrfAnswer struct {
	Token string `json:"token"`
}

func TestCSRFTokenSetsCookieAndAuthorizesCookieWrites(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "browser", "Browser")
	sessionToken := environment.signIn(t, user.ID)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/csrf", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	recorder := httptest.NewRecorder()
	environment.handler.ServeHTTP(recorder, request)

	expectStatus(t, recorder, http.StatusOK)
	answer := decodeBody[csrfAnswer](t, recorder)
	var csrfCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
		}
	}
	if csrfCookie == nil || csrfCookie.Value != answer.Token || answer.Token == "" {
		t.Fatalf("token = %q, cookie = %#v", answer.Token, csrfCookie)
	}
	if !csrfCookie.HttpOnly {
		t.Fatal("csrf_token cookie is readable by scripts")
	}

	logout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	logout.AddCookie(csrfCookie)
	logout.Header.Set("X-CSRF-Token", answer.Token)
	recorder = httptest.NewRecorder()
	environment.handler.ServeHTTP(recorder, logout)
	expectStatus(t, recorder, http.StatusNoContent)
}

func TestCSRFTokenKeepsExistingCookie(t *testing.T) {
	environment := newAPIEnvironment(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/csrf", nil)
	request.AddCookie(&http.Cookie{Name: "csrf_token", Value: testCSRFToken})
	recorder := httptest.NewRecorder()
	environment.handler.ServeHTTP(recorder, request)

	expectStatus(t, recorder, http.StatusOK)
	answer := decodeBody[csrfAnswer](t, recorder)
	if answer.Token != testCSRFToken {
		t.Fatalf("token = %q, want the cookie value", answer.Token)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %#v, want none", recorder.Result().Cookies())
	}
}
