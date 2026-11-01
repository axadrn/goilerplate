package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/testutil"
)

func seedOperator(t *testing.T, environment *apiEnvironment) string {
	t.Helper()
	operator := testutil.SeedUser(t, environment.database, testutil.UserOptions{ID: "operator", Email: testAdminEmail, Name: "Operator", Verified: true})
	return operator.User.ID
}

func TestAccountReportsAdmin(t *testing.T) {
	environment := newAPIEnvironment(t)
	operatorToken := environment.signIn(t, seedOperator(t, environment))
	target := seedVerifiedUser(t, environment, "member", "Member")
	targetToken := environment.signIn(t, target.ID)

	for _, check := range []struct {
		name  string
		token string
		want  bool
	}{
		{name: "operator", token: operatorToken, want: true},
		{name: "user", token: targetToken, want: false},
	} {
		recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: check.token})
		expectStatus(t, recorder, http.StatusOK)
		body := decodeBody[map[string]any](t, recorder)
		if body["is_admin"] != check.want {
			t.Fatalf("%s is_admin = %#v, want %v", check.name, body["is_admin"], check.want)
		}
	}
}

func TestAdminUsers(t *testing.T) {
	environment := newAPIEnvironment(t)
	operatorToken := environment.signIn(t, seedOperator(t, environment))
	target := seedVerifiedUser(t, environment, "ada", "Ada Lovelace")
	seedVerifiedUser(t, environment, "grace", "Grace Hopper")
	targetToken := environment.signIn(t, target.ID)

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/admin/users", bearer: operatorToken})
	expectStatus(t, recorder, http.StatusOK)
	dashboard := decodeBody[api.AdminDashboard](t, recorder)
	if dashboard.Metrics.Users != 3 || dashboard.Users.TotalUsers != 3 || len(dashboard.Users.Items) != 3 || dashboard.Users.Page != 1 {
		t.Fatalf("dashboard = %s", recorder.Body.String())
	}

	filtered := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/admin/users?query=ada&page=1", bearer: operatorToken})
	expectStatus(t, filtered, http.StatusOK)
	page := decodeBody[api.AdminDashboard](t, filtered).Users
	if page.Query != "ada" || len(page.Items) != 1 || page.Items[0].ID != target.ID || page.Items[0].IsAdmin {
		t.Fatalf("filtered users = %#v", page)
	}

	denied := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/admin/users", bearer: targetToken})
	expectError(t, denied, http.StatusForbidden, "forbidden")
	guest := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/admin/users"})
	expectError(t, guest, http.StatusUnauthorized, "unauthenticated")
}

func TestAdminRevokeUserSessions(t *testing.T) {
	environment := newAPIEnvironment(t)
	operatorID := seedOperator(t, environment)
	operatorToken := environment.signIn(t, operatorID)
	target := seedVerifiedUser(t, environment, "target", "Target")
	targetToken := environment.signIn(t, target.ID)

	denied := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/admin/users/" + operatorID + "/sessions/revoke", bearer: targetToken})
	expectError(t, denied, http.StatusForbidden, "forbidden")

	recorder := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/admin/users/" + target.ID + "/sessions/revoke", bearer: operatorToken})
	expectStatus(t, recorder, http.StatusNoContent)
	revoked := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: targetToken})
	expectError(t, revoked, http.StatusUnauthorized, "unauthenticated")

	self := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/admin/users/" + operatorID + "/sessions/revoke", bearer: operatorToken})
	expectError(t, self, http.StatusConflict, "admin_self_action")
	missing := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/admin/users/missing/sessions/revoke", bearer: operatorToken})
	expectError(t, missing, http.StatusNotFound, "not_found")

	dashboard := decodeBody[api.AdminDashboard](t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/admin/users", bearer: operatorToken}))
	if len(dashboard.Events) == 0 || dashboard.Events[0].Action != "admin.sessions_revoked" || dashboard.Events[0].TargetID != target.ID {
		t.Fatalf("events = %#v", dashboard.Events)
	}
}

func TestAdminRevokeUserSessionsRevokesAPITokens(t *testing.T) {
	environment := newAPIEnvironment(t)
	operatorToken := environment.signIn(t, seedOperator(t, environment))
	target := seedTokenOwner(t, environment, "compromised")
	created := createAPIToken(t, environment, target, `"name":"Script"`)

	recorder := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/admin/users/" + target.id + "/sessions/revoke", bearer: operatorToken})
	expectStatus(t, recorder, http.StatusNoContent)

	expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: created.Token}), http.StatusUnauthorized, "unauthenticated")
}

func TestImpersonationIsReadOnlyUnderAPI(t *testing.T) {
	environment := newAPIEnvironment(t)
	operatorID := seedOperator(t, environment)
	operatorToken := environment.signIn(t, operatorID)
	target := seedVerifiedUser(t, environment, "viewed", "Viewed")

	operator, err := environment.users.ByID(operatorID)
	if err != nil {
		t.Fatalf("operator lookup: %v", err)
	}
	operatorSession, err := environment.auth.AuthenticateSession(operatorToken)
	if err != nil {
		t.Fatalf("operator session: %v", err)
	}
	start := httptest.NewRecorder()
	err = environment.admin.StartImpersonation(start, httptest.NewRequest(http.MethodPost, "/admin/users/"+target.ID+"/impersonate", nil), operator, operatorSession, target.ID)
	if err != nil {
		t.Fatalf("StartImpersonation() error = %v", err)
	}
	var impersonation *http.Cookie
	for _, cookie := range start.Result().Cookies() {
		if cookie.Name == "impersonation_token" {
			impersonation = cookie
		}
	}
	if impersonation == nil {
		t.Fatal("no impersonation cookie")
	}
	cookies := []*http.Cookie{impersonation}

	account := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", cookie: operatorToken, cookies: cookies})
	expectStatus(t, account, http.StatusOK)
	if decodeBody[api.UserResponse](t, account).User.ID != target.ID {
		t.Fatalf("impersonated account = %s", account.Body.String())
	}
	if decodeBody[map[string]any](t, account)["is_admin"] != false {
		t.Fatalf("impersonating operator is reported as admin: %s", account.Body.String())
	}
	sessions := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account/sessions", cookie: operatorToken, cookies: cookies})
	expectStatus(t, sessions, http.StatusOK)

	write := environment.send(t, apiRequest{method: http.MethodPatch, path: "/api/v1/account/name", body: `{"name":"Changed"}`, cookie: operatorToken, cookies: cookies})
	expectError(t, write, http.StatusForbidden, "impersonation_read_only")
	unchanged, err := environment.users.ByID(target.ID)
	if err != nil || unchanged.Name != "Viewed" {
		t.Fatalf("name after blocked write = %q, error = %v", unchanged.Name, err)
	}

	bearer := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: impersonation.Value})
	expectError(t, bearer, http.StatusUnauthorized, "unauthenticated")
}
