package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/api"
)

// cookieSignIn sends a sign-in with ?session=cookie like the browser app:
// the csrf_token cookie, and the X-CSRF-Token header when withHeader is set.
func cookieSignIn(t *testing.T, environment *apiEnvironment, path, body string, withHeader bool, extra ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path+"?session=cookie", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "goilerplate-test/1.0")
	request.AddCookie(&http.Cookie{Name: "csrf_token", Value: testCSRFToken})
	if withHeader {
		request.Header.Set("X-CSRF-Token", testCSRFToken)
	}
	for _, cookie := range extra {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	environment.handler.ServeHTTP(recorder, request)
	return recorder
}

// expectCookieSession checks a cookie sign-in answer: an HttpOnly session
// cookie that authenticates, and a body without the token.
func expectCookieSession(t *testing.T, environment *apiEnvironment, recorder *httptest.ResponseRecorder, userID string) string {
	t.Helper()
	expectStatus(t, recorder, http.StatusOK)
	var fields map[string]json.RawMessage
	err := json.Unmarshal(recorder.Body.Bytes(), &fields)
	if err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	if len(fields) != 2 || fields["expires_at"] == nil || fields["user"] == nil {
		t.Fatalf("body = %s, want only expires_at and user", recorder.Body.String())
	}
	session := decodeBody[api.CookieSession](t, recorder)
	if session.User.ID != userID || session.ExpiresAt.IsZero() {
		t.Fatalf("session = %#v, want user %s", session, userID)
	}
	var sessionCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("no session cookie in %#v", recorder.Result().Cookies())
	}
	if !sessionCookie.HttpOnly || sessionCookie.Path != "/" {
		t.Fatalf("session cookie = %#v, want HttpOnly on /", sessionCookie)
	}
	if strings.Contains(recorder.Body.String(), sessionCookie.Value) {
		t.Fatal("the body contains the session token")
	}
	account := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", cookie: sessionCookie.Value})
	expectStatus(t, account, http.StatusOK)
	return sessionCookie.Value
}

func TestPasswordSignInWithCookieSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "browser", "Browser")
	body := `{"email":"browser@example.com","password":"` + testPassword + `"}`

	recorder := cookieSignIn(t, environment, "/api/v1/auth/password", body, true)

	expectCookieSession(t, environment, recorder, user.ID)
}

func TestCookieSignInReplacesThePreviousCookieSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "returning", "Returning")
	other := seedVerifiedUser(t, environment, "planted", "Planted")
	previous := environment.signIn(t, other.ID)
	body := `{"email":"returning@example.com","password":"` + testPassword + `"}`

	recorder := cookieSignIn(t, environment, "/api/v1/auth/password", body, true, &http.Cookie{Name: "session_token", Value: previous})

	expectCookieSession(t, environment, recorder, user.ID)
	stale := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: previous})
	expectError(t, stale, http.StatusUnauthorized, "unauthenticated")
}

func TestCookieSessionSignInNeedsTheCSRFHeader(t *testing.T) {
	environment := newAPIEnvironment(t)
	seedVerifiedUser(t, environment, "browser", "Browser")
	body := `{"email":"browser@example.com","password":"` + testPassword + `"}`

	withoutHeader := cookieSignIn(t, environment, "/api/v1/auth/password", body, false)
	detail := expectError(t, withoutHeader, http.StatusForbidden, "forbidden")
	if detail.Message != "Send the X-CSRF-Token header with cookie requests." {
		t.Fatalf("message = %q", detail.Message)
	}
	if cookieValue(withoutHeader, "session_token") != "" {
		t.Fatal("a rejected cookie sign-in set the session cookie")
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password?session=cookie", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", testCSRFToken)
	withoutCookie := httptest.NewRecorder()
	environment.handler.ServeHTTP(withoutCookie, request)
	expectError(t, withoutCookie, http.StatusForbidden, "forbidden")

	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/password?session=cookie", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer anything")
	withAuthorization := httptest.NewRecorder()
	environment.handler.ServeHTTP(withAuthorization, request)
	expectError(t, withAuthorization, http.StatusForbidden, "forbidden")
	if cookieValue(withAuthorization, "session_token") != "" {
		t.Fatal("an Authorization header skipped the CSRF check of a cookie sign-in")
	}
}

func TestBearerSignInIsUnchanged(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "script", "Script")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", strings.NewReader(`{"email":"script@example.com","password":"`+testPassword+`"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	environment.handler.ServeHTTP(recorder, request)

	expectStatus(t, recorder, http.StatusOK)
	expectSession(t, recorder, decodeBody[api.SessionToken](t, recorder), user.ID)
}

func TestMagicLinkVerifyWithCookieSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "linked", "Linked")
	err := environment.auth.SendMagicLink("linked@example.com")
	if err != nil {
		t.Fatalf("SendMagicLink() error = %v", err)
	}
	body := `{"token":"` + environment.mailedToken(t, "/auth/magic-link/") + `"}`

	rejected := cookieSignIn(t, environment, "/api/v1/auth/magic-link/verify", body, false)
	expectError(t, rejected, http.StatusForbidden, "forbidden")

	recorder := cookieSignIn(t, environment, "/api/v1/auth/magic-link/verify", body, true)
	expectCookieSession(t, environment, recorder, user.ID)
}

func TestForgotPasswordVerifyWithCookieSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "forgetful", "Forgetful")
	err := environment.auth.SendForgotPasswordLink("forgetful@example.com")
	if err != nil {
		t.Fatalf("SendForgotPasswordLink() error = %v", err)
	}
	body := `{"token":"` + environment.mailedToken(t, "/auth/forgot-password/") + `"}`

	rejected := cookieSignIn(t, environment, "/api/v1/auth/forgot-password/verify", body, false)
	expectError(t, rejected, http.StatusForbidden, "forbidden")

	recorder := cookieSignIn(t, environment, "/api/v1/auth/forgot-password/verify", body, true)
	expectCookieSession(t, environment, recorder, user.ID)
}

func TestEmailChangeVerifyWithCookieSession(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "moving", "Moving")
	err := environment.auth.RequestEmailChange(user.ID, "moved@example.com")
	if err != nil {
		t.Fatalf("RequestEmailChange() error = %v", err)
	}
	body := `{"token":"` + environment.mailedToken(t, "/auth/verify-email-change/") + `"}`
	oldSession := environment.signIn(t, user.ID)
	oldCookie := &http.Cookie{Name: "session_token", Value: oldSession}

	rejected := cookieSignIn(t, environment, "/api/v1/auth/email-change/verify", body, false, oldCookie)
	expectError(t, rejected, http.StatusForbidden, "forbidden")

	recorder := cookieSignIn(t, environment, "/api/v1/auth/email-change/verify", body, true, oldCookie)
	newSession := expectCookieSession(t, environment, recorder, user.ID)
	sessionCookies := 0
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookies++
		}
	}
	if sessionCookies != 1 || newSession == oldSession {
		t.Fatalf("session cookies = %d, new session reused the old one: %t", sessionCookies, newSession == oldSession)
	}
	stale := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", cookie: oldSession})
	expectError(t, stale, http.StatusUnauthorized, "unauthenticated")
}
