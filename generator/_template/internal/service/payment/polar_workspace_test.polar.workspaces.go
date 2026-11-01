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

func TestPolarStatusPreservesVisibleBillingStates(t *testing.T) {
	provider := &PolarProvider{}
	tests := map[string]string{
		"active": model.SubscriptionStatusActive, "trialing": model.SubscriptionStatusTrialing,
		"past_due": model.SubscriptionStatusPastDue, "incomplete": model.SubscriptionStatusIncomplete,
		"canceled": model.SubscriptionStatusCancelled, "revoked": model.SubscriptionStatusInactive,
		"unpaid": model.SubscriptionStatusInactive, "incomplete_expired": model.SubscriptionStatusInactive,
	}
	for input, want := range tests {
		if got := provider.mapPolarStatus(input); got != want {
			t.Fatalf("mapPolarStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPolarRejectsInvalidWebhookSignature(t *testing.T) {
	provider := &PolarProvider{cfg: &config.Config{PolarWebhookSecret: "webhook-secret"}}
	headers := http.Header{
		"webhook-id": {"event"}, "webhook-timestamp": {"0"}, "webhook-signature": {"invalid"},
	}
	err := provider.HandleWebhook([]byte(`{"type":"subscription.created","data":{}}`), headers)
	if err == nil || !strings.Contains(err.Error(), "invalid webhook signature") {
		t.Fatalf("HandleWebhook() error = %v, want signature verification error", err)
	}
}

func TestPolarWebhookBindsSubscriptionToWorkspace(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "polar-workspace"})
	subscriptions := service.NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	provider := NewPolarProvider(&config.Config{PolarSandboxMode: true}, subscriptions)
	payload, _ := json.Marshal(map[string]any{
		"id": "polar-subscription-1", "customer_id": "polar-customer-1", "status": "active",
		"metadata": map[string]string{"workspace_id": fixture.Workspace.ID, "plan_id": model.SubscriptionPlanPro},
	})

	if err := provider.handleSubscriptionCreated(payload); err != nil {
		t.Fatalf("handleSubscriptionCreated() error = %v", err)
	}
	subscription, err := subscriptions.Subscription(fixture.Workspace.ID)
	if err != nil {
		t.Fatalf("Subscription() error = %v", err)
	}
	if subscription.Provider != model.ProviderPolar || subscription.PlanID != model.SubscriptionPlanPro || stringValue(subscription.ProviderCustomerID) != "polar-customer-1" {
		t.Fatalf("workspace subscription = %#v", subscription)
	}
}

func TestPolarIncompleteSubscriptionStaysInactive(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "polar-incomplete"})
	subscriptions := service.NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	provider := NewPolarProvider(&config.Config{PolarSandboxMode: true}, subscriptions)
	payload, _ := json.Marshal(map[string]any{
		"id": "polar-incomplete", "customer_id": "customer-incomplete", "status": "incomplete",
		"metadata": map[string]string{"workspace_id": fixture.Workspace.ID, "plan_id": model.SubscriptionPlanPro},
	})
	if err := provider.handleSubscriptionCreated(payload); err != nil {
		t.Fatalf("handleSubscriptionCreated() error = %v", err)
	}
	subscription, err := subscriptions.Subscription(fixture.Workspace.ID)
	if err != nil {
		t.Fatalf("Subscription() error = %v", err)
	}
	if subscription.Status != model.SubscriptionStatusIncomplete || subscription.IsActive() {
		t.Fatalf("incomplete subscription = %#v, want inactive", subscription)
	}
}
