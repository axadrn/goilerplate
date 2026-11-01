package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFProtectionSkipsRequestsWithAuthorization(t *testing.T) {
	called := false
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer token")
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: generateCSRFToken()})
	recorder := httptest.NewRecorder()

	CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("called = %t, status = %d", called, recorder.Code)
	}
}

func TestCSRFProtectionChecksOtherAuthorizationSchemes(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/logout", "/auth/logout"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		request.AddCookie(&http.Cookie{Name: "session_token", Value: "session"})
		request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: generateCSRFToken()})
		recorder := httptest.NewRecorder()

		CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("%s: handler was called without a CSRF token", path)
		})).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s: status = %d, want 403", path, recorder.Code)
		}
	}
}

func TestCSRFProtectionRejectsAPICookieRequestWithJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: "session"})
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: generateCSRFToken()})
	recorder := httptest.NewRecorder()

	CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	detail := decodeAPIError(t, recorder)
	if detail.Code != "forbidden" {
		t.Fatalf("code = %q, want forbidden", detail.Code)
	}
	if detail.Message != "Send the X-CSRF-Token header with cookie requests." {
		t.Fatalf("message = %q", detail.Message)
	}
}

func TestCSRFProtectionSkipsAPIRequestsWithoutSessionCookie(t *testing.T) {
	called := false
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", nil)
	recorder := httptest.NewRecorder()

	CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("called = %t, status = %d", called, recorder.Code)
	}
}

// TestCSRFProtectionChecksCookieSignIn keeps the cookie-less API exemption
// away from sign-ins that ask for a session cookie, even with an
// Authorization header.
func TestCSRFProtectionChecksCookieSignIn(t *testing.T) {
	token := generateCSRFToken()
	tests := []struct {
		name          string
		header        string
		authorization string
		wantCalled    bool
	}{
		{name: "without header", wantCalled: false},
		{name: "with Authorization", authorization: "Bearer token", wantCalled: false},
		{name: "wrong header", header: generateCSRFToken(), wantCalled: false},
		{name: "with header", header: token, wantCalled: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password?session=cookie", nil)
			request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
			if test.header != "" {
				request.Header.Set(csrfHeader, test.header)
			}
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			recorder := httptest.NewRecorder()

			CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(recorder, request)

			if called != test.wantCalled {
				t.Fatalf("called = %t, want %t, status %d", called, test.wantCalled, recorder.Code)
			}
			if !test.wantCalled && recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", recorder.Code)
			}
		})
	}
}

func TestCSRFProtectionKeepsCheckingHTMLRequestsWithoutSessionCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/password", nil)
	recorder := httptest.NewRecorder()

	CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestRequestEmailReadsJSONAndRestoresBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", strings.NewReader(`{"email":" Ada@Example.com "}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	email := requestEmail(request)
	if email != "ada@example.com" {
		t.Fatalf("requestEmail() = %q", email)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(body) != `{"email":" Ada@Example.com "}` {
		t.Fatalf("restored body = %q", body)
	}
}

func TestRequestEmailReadsForms(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader("email=Ada%40Example.com"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	email := requestEmail(request)
	if email != "ada@example.com" {
		t.Fatalf("requestEmail() = %q", email)
	}
}
