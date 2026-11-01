package service

import (
	"testing"
	"time"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestCreateFreeSubscriptionPersistsActivePlan(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "free-subscription-user"})
	subscriptionRepository := repository.NewSubscriptionRepository(database)
	subscriptionService := NewSubscriptionService(subscriptionRepository)

	if _, err := database.Exec(`DELETE FROM subscriptions WHERE id = $1`, fixture.Subscription.ID); err != nil {
		t.Fatalf("delete fixture subscription: %v", err)
	}
	if err := subscriptionService.CreateFreeSubscription(fixture.User.ID); err != nil {
		t.Fatalf("CreateFreeSubscription() error = %v", err)
	}

	subscription, err := subscriptionService.Subscription(fixture.User.ID)
	if err != nil {
		t.Fatalf("Subscription() error = %v", err)
	}
	if subscription.PlanID != model.SubscriptionPlanFree || subscription.Status != model.SubscriptionStatusActive {
		t.Fatalf("subscription = %#v", subscription)
	}
	if subscription.Provider != "" || subscription.ProviderCustomerID != nil || subscription.ProviderSubscriptionID != nil {
		t.Fatalf("free subscription contains provider state: %#v", subscription)
	}
}

func TestEndPaidAccessClearsProviderSubscriptionState(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "downgrade-user"})
	subscriptionRepository := repository.NewSubscriptionRepository(database)
	subscriptionService := NewSubscriptionService(subscriptionRepository)

	providerCustomerID := "customer-1"
	providerSubscriptionID := "subscription-1"
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	amount := 2900
	interval := model.SubscriptionIntervalMonthly
	fixture.Subscription.PlanID = model.SubscriptionPlanPro
	fixture.Subscription.Status = model.SubscriptionStatusCancelled
	fixture.Subscription.Provider = model.ProviderStripe
	fixture.Subscription.ProviderCustomerID = &providerCustomerID
	fixture.Subscription.ProviderSubscriptionID = &providerSubscriptionID
	fixture.Subscription.CurrentPeriodEnd = &periodEnd
	fixture.Subscription.Amount = &amount
	fixture.Subscription.Currency = "usd"
	fixture.Subscription.Interval = &interval
	if err := subscriptionRepository.Update(fixture.Subscription); err != nil {
		t.Fatalf("persist paid fixture: %v", err)
	}

	if err := subscriptionService.EndPaidAccess(fixture.Subscription); err != nil {
		t.Fatalf("EndPaidAccess() error = %v", err)
	}

	subscription, err := subscriptionRepository.ByUserID(fixture.User.ID)
	if err != nil {
		t.Fatalf("subscription lookup: %v", err)
	}
	if subscription.PlanID != model.SubscriptionPlanPro || subscription.Status != model.SubscriptionStatusInactive {
		t.Fatalf("subscription = %#v", subscription)
	}
	if subscription.Access().State != access.StateFree {
		t.Fatalf("effective access = %#v, want free fallback", subscription.Access())
	}
	if subscription.ProviderSubscriptionID != nil || subscription.CurrentPeriodEnd != nil || subscription.Amount != nil || subscription.Currency != "" || subscription.Interval != nil {
		t.Fatalf("downgraded subscription retains paid state: %#v", subscription)
	}
	if subscription.Provider != model.ProviderStripe || subscription.ProviderCustomerID == nil || *subscription.ProviderCustomerID != providerCustomerID {
		t.Fatalf("downgraded subscription lost reusable customer state: %#v", subscription)
	}
}
