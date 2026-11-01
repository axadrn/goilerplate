package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/ctxkeys"
)

func TestCSRFProtectionIssuesTokenOnSafeRequest(t *testing.T) {
	var contextToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextToken = ctxkeys.CSRFToken(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/app/acme/settings", nil)

	CSRFProtection(false, NewClientIPResolver(nil))(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent || contextToken == "" {
		t.Fatalf("status = %d, context token = %q", recorder.Code, contextToken)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != csrfCookieName || cookie.Value != contextToken || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("CSRF cookie = %#v", cookie)
	}
}

func TestCSRFProtectionAcceptsHeaderAndFormTokens(t *testing.T) {
	token := generateCSRFToken()
	tests := []struct {
		name    string
		request func() *http.Request
	}{
		{
			name: "header",
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodPatch, "/app/acme/profile/name", nil)
				request.Header.Set(csrfHeader, token)
				return request
			},
		},
		{
			name: "form",
			request: func() *http.Request {
				form := url.Values{csrfFormField: {token}}
				request := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(form.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return request
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got := ctxkeys.CSRFToken(r.Context()); got != token {
					t.Fatalf("context token = %q, want %q", got, token)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			request := test.request()
			request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
			recorder := httptest.NewRecorder()

			CSRFProtection(false, NewClientIPResolver(nil))(next).ServeHTTP(recorder, request)

			if !called || recorder.Code != http.StatusNoContent {
				t.Fatalf("called = %t, status = %d", called, recorder.Code)
			}
		})
	}
}

func TestCSRFProtectionRejectsMissingOrMismatchedToken(t *testing.T) {
	for _, submittedToken := range []string{"", generateCSRFToken()} {
		request := httptest.NewRequest(http.MethodDelete, "/app/acme/settings/sessions", nil)
		request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: generateCSRFToken()})
		if submittedToken != "" {
			request.Header.Set(csrfHeader, submittedToken)
		}
		recorder := httptest.NewRecorder()

		CSRFProtection(false, NewClientIPResolver(nil))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("protected handler was called")
		})).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("submitted token %q status = %d, want %d", submittedToken, recorder.Code, http.StatusForbidden)
		}
	}
}

func TestCSRFProtectionBypassesWebhooks(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader("{}"))
	recorder := httptest.NewRecorder()

	CSRFProtection(false, NewClientIPResolver(nil))(next).ServeHTTP(recorder, request)

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("called = %t, status = %d", called, recorder.Code)
	}
}
