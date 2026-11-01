package api_test

import (
	"net/http"
	"testing"
	"time"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

// billingWorkspace seeds an owner with a workspace plus an admin, a member
// and an outsider, and returns their bearer tokens.
type billingWorkspace struct {
	workspace *model.Workspace
	// ownerMemberID is the member row of the owner. The admin and member
	// rows are "member-admin" and "member-member".
	ownerMemberID string
	owner         string
	admin         string
	member        string
	outsider      string
}

func seedBillingWorkspace(t *testing.T, environment *apiEnvironment) billingWorkspace {
	t.Helper()
	fixture := testutil.SeedUser(t, environment.database, testutil.UserOptions{ID: "owner", Email: "owner@example.com", Name: "Owner", Verified: true})
	seeded := billingWorkspace{workspace: fixture.Workspace, ownerMemberID: fixture.WorkspaceMember.ID, owner: environment.signIn(t, fixture.User.ID)}
	for _, role := range []string{model.WorkspaceRoleAdmin, model.WorkspaceRoleMember} {
		user := testutil.SeedUser(t, environment.database, testutil.UserOptions{ID: role + "-user", Email: role + "@example.com", Name: role, Verified: true})
		_, err := environment.database.Exec(`
			INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, "member-"+role, fixture.Workspace.ID, user.User.ID, role, time.Now())
		if err != nil {
			t.Fatalf("add %s: %v", role, err)
		}
		if role == model.WorkspaceRoleAdmin {
			seeded.admin = environment.signIn(t, user.User.ID)
		} else {
			seeded.member = environment.signIn(t, user.User.ID)
		}
	}
	outsider := testutil.SeedUser(t, environment.database, testutil.UserOptions{ID: "outsider", Email: "outsider@example.com", Name: "Outsider", Verified: true})
	seeded.outsider = environment.signIn(t, outsider.User.ID)
	return seeded
}

func TestWorkspaceBillingPermissions(t *testing.T) {
	environment := newAPIEnvironment(t)
	seeded := seedBillingWorkspace(t, environment)
	base := "/api/v1/workspaces/" + seeded.workspace.Slug + "/billing"

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: base, bearer: seeded.owner})
	expectStatus(t, recorder, http.StatusOK)
	billing := decodeBody[api.Billing](t, recorder)
	if billing.Subscription == nil || billing.Access == nil || billing.Access.PlanID != model.SubscriptionPlanFree {
		t.Fatalf("billing = %s", recorder.Body.String())
	}
	expectPurchasablePlans(t, billing.Plans)

	for _, request := range []apiRequest{
		{method: http.MethodGet, path: base},
		{method: http.MethodPost, path: base + "/checkout", body: `{"plan_id":"pro"}`},
		{method: http.MethodGet, path: base + "/manage"},
	} {
		for name, token := range map[string]string{"admin": seeded.admin, "member": seeded.member} {
			request.bearer = token
			denied := environment.send(t, request)
			if denied.Code != http.StatusForbidden {
				t.Fatalf("%s %s %s: status %d", name, request.method, request.path, denied.Code)
			}
			expectError(t, denied, http.StatusForbidden, "forbidden")
		}
		request.bearer = seeded.outsider
		expectError(t, environment.send(t, request), http.StatusNotFound, "not_found")
		request.bearer = ""
		expectError(t, environment.send(t, request), http.StatusUnauthorized, "unauthenticated")
	}
	if len(environment.payments.checkoutCalls()) != 0 {
		t.Fatal("a denied checkout reached the provider")
	}

	unknown := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/workspaces/missing/billing", bearer: seeded.owner})
	expectError(t, unknown, http.StatusNotFound, "not_found")
}

func TestWorkspaceBillingCheckout(t *testing.T) {
	environment := newAPIEnvironment(t)
	seeded := seedBillingWorkspace(t, environment)
	path := "/api/v1/workspaces/" + seeded.workspace.Slug + "/billing/checkout"

	recorder := environment.send(t, apiRequest{method: http.MethodPost, path: path, body: `{"plan_id":"enterprise","interval":"yearly"}`, bearer: seeded.owner})
	expectStatus(t, recorder, http.StatusOK)
	if decodeBody[api.URLResponse](t, recorder).URL != "https://checkout.example.com/session" {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	calls := environment.payments.checkoutCalls()
	want := checkoutCall{workspaceID: seeded.workspace.ID, workspaceSlug: seeded.workspace.Slug, planID: "enterprise", interval: "yearly", email: "owner@example.com", name: seeded.workspace.Name}
	if len(calls) != 1 || calls[0] != want {
		t.Fatalf("checkout calls = %#v", calls)
	}
	var events int
	err := environment.database.Get(&events, `SELECT COUNT(*) FROM workspace_audit_events WHERE workspace_id = $1 AND action = $2`, seeded.workspace.ID, "billing.checkout_started")
	if err != nil || events != 1 {
		t.Fatalf("checkout audit events = %d, error = %v", events, err)
	}

	for _, test := range []struct {
		body string
		code string
	}{
		{body: `{"plan_id":"free"}`, code: "invalid_plan"},
		{body: `{"plan_id":"pro","interval":"weekly"}`, code: "invalid_interval"},
	} {
		rejected := environment.send(t, apiRequest{method: http.MethodPost, path: path, body: test.body, bearer: seeded.owner})
		expectError(t, rejected, http.StatusUnprocessableEntity, test.code)
	}

	environment.payments.checkoutErr = repository.ErrCheckoutPending
	pending := environment.send(t, apiRequest{method: http.MethodPost, path: path, body: `{"plan_id":"pro"}`, bearer: seeded.owner})
	expectError(t, pending, http.StatusConflict, "checkout_pending")
	environment.payments.checkoutErr = repository.ErrSubscriptionAlreadyPaid
	paid := environment.send(t, apiRequest{method: http.MethodPost, path: path, body: `{"plan_id":"pro"}`, bearer: seeded.owner})
	expectError(t, paid, http.StatusConflict, "already_paid")
}

func TestWorkspaceBillingManagement(t *testing.T) {
	environment := newAPIEnvironment(t)
	seeded := seedBillingWorkspace(t, environment)
	path := "/api/v1/workspaces/" + seeded.workspace.Slug + "/billing/manage"

	free := environment.send(t, apiRequest{method: http.MethodGet, path: path, bearer: seeded.owner})
	expectError(t, free, http.StatusConflict, "no_billing_account")

	_, err := environment.database.Exec(`UPDATE subscriptions SET provider_customer_id = $1 WHERE workspace_id = $2`, "customer-1", seeded.workspace.ID)
	if err != nil {
		t.Fatalf("set provider customer: %v", err)
	}
	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: path, bearer: seeded.owner})
	expectStatus(t, recorder, http.StatusOK)
	if decodeBody[api.URLResponse](t, recorder).URL != "https://billing.example.com/portal" {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if len(environment.payments.portalCalls) != 1 || environment.payments.portalCalls[0] != seeded.workspace.ID {
		t.Fatalf("portal calls = %#v", environment.payments.portalCalls)
	}
}
