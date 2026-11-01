package service

import (
	"errors"
	"sync"
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
	subscriptionService := NewSubscriptionService(subscriptionRepository, repository.NewAdminRepository(database))

	if _, err := database.Exec(`DELETE FROM subscriptions WHERE id = $1`, fixture.Subscription.ID); err != nil {
		t.Fatalf("delete fixture subscription: %v", err)
	}
	if err := subscriptionService.CreateFreeSubscription(fixture.Workspace.ID); err != nil {
		t.Fatalf("CreateFreeSubscription() error = %v", err)
	}

	subscription, err := subscriptionService.Subscription(fixture.Workspace.ID)
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

func TestEnsureSubscriptionRestoresOneFreeRowConcurrently(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "ensure-subscription-user"})
	subscriptionService := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	if _, err := database.Exec(`DELETE FROM subscriptions WHERE workspace_id = $1`, fixture.Workspace.ID); err != nil {
		t.Fatalf("delete fixture subscription: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 8)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			subscription, err := subscriptionService.EnsureSubscription(fixture.Workspace.ID)
			if err == nil && (subscription.PlanID != model.SubscriptionPlanFree || subscription.Status != model.SubscriptionStatusActive) {
				err = errors.New("restored subscription is not active Free")
			}
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("EnsureSubscription() error = %v", err)
		}
	}

	var count int
	if err := database.Get(&count, `SELECT COUNT(*) FROM subscriptions WHERE workspace_id = $1`, fixture.Workspace.ID); err != nil {
		t.Fatalf("count restored subscriptions: %v", err)
	}
	if count != 1 {
		t.Fatalf("restored subscription count = %d, want 1", count)
	}
}

func TestEndPaidAccessClearsProviderSubscriptionState(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "downgrade-user"})
	subscriptionRepository := repository.NewSubscriptionRepository(database)
	subscriptionService := NewSubscriptionService(subscriptionRepository, repository.NewAdminRepository(database))

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

	subscription, err := subscriptionRepository.ByWorkspaceID(fixture.Workspace.ID)
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
	if err := subscriptionService.UpdateSubscription(subscription); err != nil {
		t.Fatalf("repeated UpdateSubscription() error = %v", err)
	}
	events, err := repository.NewAdminRepository(database).ListAuditEvents(10)
	if err != nil {
		t.Fatalf("ListAuditEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Action != "subscription.updated" || events[0].WorkspaceID == nil || *events[0].WorkspaceID != fixture.Workspace.ID {
		t.Fatalf("subscription audit events = %#v", events)
	}
}

func TestProviderEventIDsPreserveRepeatedSubscriptionTransitions(t *testing.T) {
	for _, provider := range []string{model.ProviderStripe, model.ProviderPolar} {
		t.Run(provider, func(t *testing.T) {
			database := testutil.NewDatabase(t)
			fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "audit-transition-" + provider})
			subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
			providerSubscriptionID := "subscription-" + provider
			periodEnd := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
			fixture.Subscription.Provider = provider
			fixture.Subscription.ProviderSubscriptionID = &providerSubscriptionID
			fixture.Subscription.PlanID = model.SubscriptionPlanPro
			fixture.Subscription.CurrentPeriodEnd = &periodEnd

			for _, transition := range []struct {
				eventID string
				status  string
			}{
				{"event-active-1", model.SubscriptionStatusActive},
				{"event-cancelled", model.SubscriptionStatusCancelled},
				{"event-active-2", model.SubscriptionStatusActive},
				{"event-active-2", model.SubscriptionStatusActive},
			} {
				fixture.Subscription.Status = transition.status
				if err := subscriptions.UpdateSubscription(fixture.Subscription, transition.eventID); err != nil {
					t.Fatalf("UpdateSubscription(%s) error = %v", transition.eventID, err)
				}
			}

			events, err := repository.NewAdminRepository(database).ListAuditEvents(10)
			if err != nil {
				t.Fatalf("ListAuditEvents() error = %v", err)
			}
			if len(events) != 3 {
				t.Fatalf("subscription audit event count = %d, want 3", len(events))
			}
		})
	}
}

func TestCheckoutStateRejectsConcurrentCreationAndReusesOpenURL(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "checkout-marker-user"})
	repo := repository.NewSubscriptionRepository(database)
	subscriptions := NewSubscriptionService(repo, repository.NewAdminRepository(database))
	expiresAt := time.Now().Add(time.Hour)
	pendingUntil := expiresAt.Add(checkoutWebhookTestGrace)
	checkoutURL, attemptID, reused, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil)
	if err != nil || checkoutURL != "" || attemptID == "" || reused {
		t.Fatalf("first BeginCheckout() URL = %q, attempt = %q, reused = %t, error = %v", checkoutURL, attemptID, reused, err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutPending) {
		t.Fatalf("concurrent BeginCheckout() error = %v, want ErrCheckoutPending", err)
	}
	if err := subscriptions.FinishCheckout(fixture.Workspace.ID, attemptID, "", expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutURLMissing) {
		t.Fatalf("empty FinishCheckout() error = %v, want ErrCheckoutURLMissing", err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutPending) {
		t.Fatalf("retry after empty checkout URL error = %v, want ErrCheckoutPending", err)
	}
	if err := subscriptions.FinishCheckout(fixture.Workspace.ID, attemptID, "https://checkout.example.com/open", expiresAt, pendingUntil); err != nil {
		t.Fatalf("FinishCheckout() error = %v", err)
	}
	reusedURL, _, reused, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil)
	if err != nil || reusedURL != "https://checkout.example.com/open" || !reused {
		t.Fatalf("reused checkout URL = %q, reused = %t, error = %v", reusedURL, reused, err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanEnterprise, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutSelection) {
		t.Fatalf("different plan error = %v, want ErrCheckoutSelection", err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalYearly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutSelection) {
		t.Fatalf("different interval error = %v, want ErrCheckoutSelection", err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderStripe, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrCheckoutSelection) {
		t.Fatalf("different provider error = %v, want ErrCheckoutSelection", err)
	}
	stored, err := repo.ByWorkspaceID(fixture.Workspace.ID)
	if err != nil {
		t.Fatalf("subscription lookup: %v", err)
	}

	providerSubscriptionID := "checkout-marker-subscription"
	stateTime := time.Now()
	stored.Provider = model.ProviderPolar
	stored.ProviderSubscriptionID = &providerSubscriptionID
	stored.PlanID = model.SubscriptionPlanPro
	stored.ProviderSubscriptionCreatedAt = &stateTime
	stored.ProviderStateUpdatedAt = &stateTime
	if updated, err := subscriptions.UpdateProviderState(stored, providerSubscriptionID, stateTime, &stateTime); err != nil || !updated {
		t.Fatalf("UpdateProviderState() updated = %t, error = %v", updated, err)
	}
	stored, err = repo.ByWorkspaceID(fixture.Workspace.ID)
	if err != nil || stored.CheckoutPendingUntil != nil {
		t.Fatalf("provider state retained checkout marker: %#v, error = %v", stored, err)
	}
	if _, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderPolar, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil); !errors.Is(err, repository.ErrSubscriptionAlreadyPaid) {
		t.Fatalf("paid BeginCheckout() error = %v, want ErrSubscriptionAlreadyPaid", err)
	}
	workspaceID := fixture.Workspace.ID
	deleteEvent := &model.AdminAuditEvent{
		ID:          "delete-event-checkout-marker",
		WorkspaceID: &workspaceID,
		Action:      "workspace.deleted",
		TargetType:  "workspace",
		TargetID:    workspaceID,
		Metadata:    "{}",
		CreatedAt:   time.Now(),
	}
	if err := repository.NewWorkspaceRepository(database).DeleteIfBillingInactive(fixture.Workspace.ID, time.Now(), deleteEvent); !errors.Is(err, repository.ErrWorkspaceBillingActive) {
		t.Fatalf("paid DeleteIfBillingInactive() error = %v, want ErrWorkspaceBillingActive", err)
	}
	if err := subscriptions.EndPaidAccess(stored); err != nil {
		t.Fatalf("EndPaidAccess() error = %v", err)
	}
	if err := repository.NewWorkspaceRepository(database).DeleteIfBillingInactive(fixture.Workspace.ID, time.Now(), deleteEvent); err != nil {
		t.Fatalf("DeleteIfBillingInactive() after termination error = %v", err)
	}
}

func TestLocalTrialCanStartProviderCheckout(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "trial-checkout-user"})
	trialEndsAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := database.Exec(`
		UPDATE subscriptions
		SET plan_id = $1, status = $2, current_period_end = $3,
		    provider_customer_id = NULL, provider_subscription_id = NULL
		WHERE workspace_id = $4
	`, model.SubscriptionPlanPro, model.SubscriptionStatusTrialing, trialEndsAt, fixture.Workspace.ID); err != nil {
		t.Fatal(err)
	}

	subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	expiresAt := time.Now().Add(time.Hour)
	_, attemptID, reused, err := subscriptions.BeginCheckout(
		fixture.Workspace.ID,
		model.ProviderStripe,
		model.SubscriptionPlanPro,
		model.SubscriptionIntervalMonthly,
		expiresAt,
		expiresAt.Add(checkoutWebhookTestGrace),
	)
	if err != nil || attemptID == "" || reused {
		t.Fatalf("BeginCheckout() attempt = %q, reused = %t, error = %v", attemptID, reused, err)
	}
}

const checkoutWebhookTestGrace = 4 * 24 * time.Hour

func TestConcurrentCheckoutCreationAllowsOneProviderAttempt(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "concurrent-checkout-user"})
	subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	expiresAt := time.Now().Add(time.Hour)
	pendingUntil := expiresAt.Add(checkoutWebhookTestGrace)
	start := make(chan struct{})
	results := make(chan error, 12)
	var wait sync.WaitGroup

	for range 12 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, _, _, err := subscriptions.BeginCheckout(fixture.Workspace.ID, model.ProviderStripe, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, expiresAt, pendingUntil)
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	successes := 0
	pending := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, repository.ErrCheckoutPending):
			pending++
		default:
			t.Fatalf("BeginCheckout() error = %v", err)
		}
	}
	if successes != 1 || pending != 11 {
		t.Fatalf("successful attempts = %d, pending attempts = %d", successes, pending)
	}
}
