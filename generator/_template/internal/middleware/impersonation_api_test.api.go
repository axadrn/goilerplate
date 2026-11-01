package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

func TestBlockImpersonatedWritesUnderAPI(t *testing.T) {
	operatorID := "operator"
	impersonation := &model.Session{ImpersonatorUserID: &operatorID}

	for _, test := range []struct {
		method     string
		path       string
		wantCalled bool
	}{
		{http.MethodGet, "/api/v1/account", true},
		{http.MethodGet, "/api/v1/account/sessions", true},
		{http.MethodHead, "/api/v1/billing", true},
		{http.MethodGet, "/api/v1/workspaces/acme/billing", true},
		{http.MethodGet, "/api/v1/billing/portal", false},
		{http.MethodGet, "/api/v1/workspaces/acme/billing/manage", false},
		{http.MethodPost, "/api/v1/billing/checkout", false},
		{http.MethodPatch, "/api/v1/account/name", false},
		{http.MethodDelete, "/api/v1/account", false},
		{http.MethodDelete, "/api/v1/account/sessions", false},
		{http.MethodPost, "/api/v1/admin/users/user-1/sessions/revoke", false},
		{http.MethodPost, "/api/v1/auth/logout", true},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request = request.WithContext(ctxkeys.WithSession(request.Context(), impersonation))
			called := false
			recorder := httptest.NewRecorder()
			BlockImpersonatedWrites(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(recorder, request)

			if called != test.wantCalled {
				t.Fatalf("called = %t, want %t", called, test.wantCalled)
			}
			if test.wantCalled {
				if recorder.Code != http.StatusNoContent {
					t.Fatalf("status = %d, want 204", recorder.Code)
				}
				return
			}
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", recorder.Code)
			}
			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
			}
			var body api.ErrorBody
			err := json.Unmarshal(recorder.Body.Bytes(), &body)
			if err != nil {
				t.Fatalf("decode %q: %v", recorder.Body.String(), err)
			}
			if body.Error.Code != "impersonation_read_only" {
				t.Fatalf("code = %q, want impersonation_read_only", body.Error.Code)
			}
		})
	}
}

func TestBlockImpersonatedWritesKeepsOrdinarySessions(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", nil)
	request = request.WithContext(ctxkeys.WithSession(request.Context(), &model.Session{ID: "session-1"}))
	called := false
	recorder := httptest.NewRecorder()
	BlockImpersonatedWrites(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(recorder, request)
	if !called {
		t.Fatalf("ordinary session was blocked: %d %s", recorder.Code, recorder.Body.String())
	}
}
