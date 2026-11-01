package payment

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/testutil"
)

func TestStripeStatusPreservesVisibleBillingStates(t *testing.T) {
	provider := &StripeProvider{}
	tests := map[string]string{
		"active": model.SubscriptionStatusActive, "trialing": model.SubscriptionStatusTrialing,
		"past_due": model.SubscriptionStatusPastDue, "canceled": model.SubscriptionStatusCancelled,
		"unpaid": model.SubscriptionStatusInactive, "incomplete_expired": model.SubscriptionStatusInactive,
	}
	for input, want := range tests {
		if got := provider.mapStripeStatus(input); got != want {
			t.Fatalf("mapStripeStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStripeRejectsInvalidWebhookSignature(t *testing.T) {
	provider := &StripeProvider{cfg: &config.Config{StripeWebhookSecret: "webhook-secret"}}
	err := provider.HandleWebhook([]byte(`{"id":"event"}`), http.Header{"Stripe-Signature": {"invalid"}})
	if err == nil || !strings.Contains(err.Error(), "verify webhook signature") {
		t.Fatalf("HandleWebhook() error = %v, want signature verification error", err)
	}
}

func TestStripeWebhookBindsCustomerToWorkspace(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "stripe-workspace"})
	subscriptions := service.NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	provider := &StripeProvider{cfg: &config.Config{}, subscriptionService: subscriptions}
	payload, _ := json.Marshal(map[string]any{
		"id":       "checkout-1",
		"customer": "stripe-customer-1",
		"metadata": map[string]string{"workspace_id": fixture.Workspace.ID},
	})

	if err := provider.handleCheckoutSessionCompleted(payload); err != nil {
		t.Fatalf("handleCheckoutSessionCompleted() error = %v", err)
	}
	subscription, err := subscriptions.Subscription(fixture.Workspace.ID)
	if err != nil {
		t.Fatalf("Subscription() error = %v", err)
	}
	if subscription.Provider != model.ProviderStripe || stringValue(subscription.ProviderCustomerID) != "stripe-customer-1" {
		t.Fatalf("workspace subscription = %#v", subscription)
	}
}
