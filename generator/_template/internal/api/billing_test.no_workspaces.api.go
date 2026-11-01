package api_test

import (
	"errors"
	"net/http"
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/model"
)

func TestBillingSummary(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "payer", "Payer")
	token := environment.signIn(t, user.ID)

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/billing", bearer: token})
	expectStatus(t, recorder, http.StatusOK)
	billing := decodeBody[api.Billing](t, recorder)
	if billing.Subscription == nil || billing.Subscription.PlanID != model.SubscriptionPlanFree {
		t.Fatalf("subscription = %#v", billing.Subscription)
	}
	if billing.Access == nil || billing.Access.PlanID != model.SubscriptionPlanFree {
		t.Fatalf("access = %#v", billing.Access)
	}
	expectPurchasablePlans(t, billing.Plans)

	guest := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/billing"})
	expectError(t, guest, http.StatusUnauthorized, "unauthenticated")
}

func TestBillingCheckout(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "buyer", "Buyer")
	token := environment.signIn(t, user.ID)

	recorder := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/billing/checkout", body: `{"plan_id":"pro","interval":"yearly"}`, bearer: token})
	expectStatus(t, recorder, http.StatusOK)
	if decodeBody[api.URLResponse](t, recorder).URL != "https://checkout.example.com/session" {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	monthly := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/billing/checkout", body: `{"plan_id":"enterprise"}`, bearer: token})
	expectStatus(t, monthly, http.StatusOK)
	calls := environment.payments.checkoutCalls()
	if len(calls) != 2 {
		t.Fatalf("checkout calls = %#v", calls)
	}
	if calls[0] != (checkoutCall{userID: user.ID, planID: "pro", interval: "yearly", email: user.Email, name: user.Name}) {
		t.Fatalf("first checkout = %#v", calls[0])
	}
	if calls[1].planID != "enterprise" || calls[1].interval != model.SubscriptionIntervalMonthly {
		t.Fatalf("second checkout = %#v", calls[1])
	}

	for _, test := range []struct {
		name string
		body string
		code string
	}{
		{name: "unknown plan", body: `{"plan_id":"platinum"}`, code: "invalid_plan"},
		{name: "free plan", body: `{"plan_id":"free"}`, code: "invalid_plan"},
		{name: "missing plan", body: `{}`, code: "invalid_plan"},
		{name: "interval", body: `{"plan_id":"pro","interval":"weekly"}`, code: "invalid_interval"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rejected := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/billing/checkout", body: test.body, bearer: token})
			expectError(t, rejected, http.StatusUnprocessableEntity, test.code)
		})
	}
	if len(environment.payments.checkoutCalls()) != 2 {
		t.Fatal("an invalid selection reached the provider")
	}

	environment.payments.checkoutErr = errors.New("provider unavailable")
	failed := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/billing/checkout", body: `{"plan_id":"pro"}`, bearer: token})
	expectError(t, failed, http.StatusInternalServerError, "internal")
}

func TestBillingPortal(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "subscriber", "Subscriber")
	token := environment.signIn(t, user.ID)

	free := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/billing/portal", bearer: token})
	expectError(t, free, http.StatusConflict, "no_billing_account")

	_, err := environment.database.Exec(`UPDATE subscriptions SET provider_customer_id = $1 WHERE user_id = $2`, "customer-1", user.ID)
	if err != nil {
		t.Fatalf("set provider customer: %v", err)
	}
	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/billing/portal", bearer: token})
	expectStatus(t, recorder, http.StatusOK)
	if decodeBody[api.URLResponse](t, recorder).URL != "https://billing.example.com/portal" {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if len(environment.payments.portalCalls) != 1 || environment.payments.portalCalls[0] != user.ID {
		t.Fatalf("portal calls = %#v", environment.payments.portalCalls)
	}
}
