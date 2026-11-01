package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v85"
	portalsession "github.com/stripe/stripe-go/v85/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/stripe/stripe-go/v85/webhook"
	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/service"
)

type StripeProvider struct {
	cfg                 *config.Config
	subscriptionService *service.SubscriptionService
}

func NewStripeProvider(cfg *config.Config, subscriptionService *service.SubscriptionService) *StripeProvider {
	// Set Stripe API key
	stripe.Key = cfg.StripeSecretKey

	slog.Info("stripe provider initialized", "app_env", cfg.AppEnv)

	return &StripeProvider{
		cfg:                 cfg,
		subscriptionService: subscriptionService,
	}
}

func (s *StripeProvider) Name() string {
	return model.ProviderStripe
}

func (s *StripeProvider) CreateCheckoutURL(workspaceID, workspaceSlug, planID, interval, customerEmail, customerName string) (string, error) {
	sub, err := s.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	priceID := s.getStripePriceID(planID, interval)
	if priceID == "" {
		return "", fmt.Errorf("no price configured for plan: %s (%s)", planID, interval)
	}
	checkoutExpiresAt := time.Now().Add(23 * time.Hour)
	checkoutURL, attemptID, reused, err := s.subscriptionService.BeginCheckout(workspaceID, s.Name(), planID, interval, checkoutExpiresAt, checkoutExpiresAt.Add(checkoutWebhookGrace))
	if err != nil {
		return "", err
	}
	if reused {
		return checkoutURL, nil
	}

	successURL := fmt.Sprintf("%s/app/%s/billing?session_id={CHECKOUT_SESSION_ID}", s.cfg.AppURL, workspaceSlug)
	cancelURL := fmt.Sprintf("%s/app/%s/billing", s.cfg.AppURL, workspaceSlug)

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(customerEmail),
		Metadata: map[string]string{
			"workspace_id":        workspaceID,
			"subscription_id":     sub.ID,
			"plan_id":             planID,
			"checkout_attempt_id": attemptID,
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{Metadata: map[string]string{
			"workspace_id":        workspaceID,
			"subscription_id":     sub.ID,
			"plan_id":             planID,
			"checkout_attempt_id": attemptID,
		}},
		AllowPromotionCodes: stripe.Bool(true),
		ExpiresAt:           stripe.Int64(checkoutExpiresAt.Unix()),
		ManagedPayments: &stripe.CheckoutSessionManagedPaymentsParams{
			Enabled: stripe.Bool(true),
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		var stripeError *stripe.Error
		if errors.As(err, &stripeError) && stripeError.HTTPStatusCode >= 400 && stripeError.HTTPStatusCode < 500 {
			s.subscriptionService.RejectCheckout(workspaceID, attemptID)
		}
		return "", fmt.Errorf("failed to create checkout session: %w", err)
	}
	if sess == nil || sess.URL == "" {
		return "", fmt.Errorf("checkout response has no URL")
	}
	if err := s.subscriptionService.FinishCheckout(workspaceID, attemptID, sess.URL, checkoutExpiresAt, checkoutExpiresAt.Add(checkoutWebhookGrace)); err != nil {
		return "", err
	}

	slog.Info("stripe checkout created", "workspace_id", workspaceID, "plan_id", planID, "session_id", sess.ID)
	return sess.URL, nil
}

func (s *StripeProvider) BillingManagementURL(workspaceID, workspaceSlug string) (string, error) {
	sub, err := s.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.ProviderCustomerID == nil || *sub.ProviderCustomerID == "" {
		return "", fmt.Errorf("no customer portal available for free subscriptions")
	}

	returnURL := fmt.Sprintf("%s/app/%s/billing", s.cfg.AppURL, workspaceSlug)

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*sub.ProviderCustomerID),
		ReturnURL: stripe.String(returnURL),
	}

	portalSession, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create customer portal session: %w", err)
	}

	slog.Info("stripe customer portal session created", "workspace_id", workspaceID)
	return portalSession.URL, nil
}

func (s *StripeProvider) HandleWebhook(payload []byte, headers http.Header) error {
	signature := headers.Get("Stripe-Signature")

	// Use ConstructEventWithOptions to ignore API version mismatch
	// Stripe's API versions are backwards compatible, so this is safe
	event, err := webhook.ConstructEventWithOptions(
		payload,
		signature,
		s.cfg.StripeWebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	slog.Info("stripe webhook received", "event_type", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutSessionCompleted(event.Data.Raw, event.ID)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(event.Data.Raw, event.ID)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(event.Data.Raw, event.ID)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(event.Data.Raw, event.ID)
	case "invoice.payment_succeeded":
		return s.handleInvoicePaymentSucceeded(event.Data.Raw, event.ID)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(event.Data.Raw, event.ID)
	default:
		slog.Warn("stripe webhook unknown event type", "event_type", event.Type)
		return nil
	}
}

func (s *StripeProvider) handleCheckoutSessionCompleted(data json.RawMessage, providerEventID ...string) error {
	var checkoutSession struct {
		ID         string            `json:"id"`
		CustomerID string            `json:"customer"`
		Metadata   map[string]string `json:"metadata"`
	}

	err := json.Unmarshal(data, &checkoutSession)
	if err != nil {
		return fmt.Errorf("failed to parse checkout session: %w", err)
	}

	workspaceID := checkoutSession.Metadata["workspace_id"]
	if workspaceID == "" {
		slog.Warn("stripe checkout session has no workspace_id in metadata, skipping")
		return nil
	}

	sub, err := s.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Store customer ID for future use
	sub.Provider = model.ProviderStripe
	sub.ProviderCustomerID = &checkoutSession.CustomerID

	err = s.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("stripe checkout completed", "workspace_id", workspaceID, "customer_id", checkoutSession.CustomerID)
	return nil
}

func (s *StripeProvider) handleSubscriptionCreated(data json.RawMessage, providerEventID ...string) error {
	var subscription struct {
		ID               string `json:"id"`
		CustomerID       string `json:"customer"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
		Items            struct {
			Data []struct {
				Price struct {
					ID         string `json:"id"`
					UnitAmount int64  `json:"unit_amount"`
					Currency   string `json:"currency"`
					Recurring  struct {
						Interval string `json:"interval"`
					} `json:"recurring"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
		Metadata map[string]string `json:"metadata"`
	}

	err := json.Unmarshal(data, &subscription)
	if err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	var sub *model.Subscription
	if workspaceID := subscription.Metadata["workspace_id"]; workspaceID != "" {
		sub, err = s.subscriptionService.Subscription(workspaceID)
	} else {
		sub, err = s.subscriptionService.ByProviderCustomerID(subscription.CustomerID)
	}
	if err != nil {
		slog.Warn("stripe subscription has unknown customer, skipping", "customer_id", subscription.CustomerID)
		return nil
	}

	if len(subscription.Items.Data) == 0 {
		return fmt.Errorf("subscription has no items")
	}

	priceID := subscription.Items.Data[0].Price.ID
	planID := s.getLocalPlanID(priceID)
	if planID != "" {
		sub.PlanID = planID
	}

	sub.Provider = model.ProviderStripe
	sub.ProviderSubscriptionID = &subscription.ID
	sub.ClearCheckout()
	sub.Status = s.mapStripeStatus(subscription.Status)

	amount := int(subscription.Items.Data[0].Price.UnitAmount)
	sub.Amount = &amount
	sub.Currency = subscription.Items.Data[0].Price.Currency

	interval := s.mapStripeInterval(subscription.Items.Data[0].Price.Recurring.Interval)
	sub.Interval = &interval

	periodEnd := time.Unix(subscription.CurrentPeriodEnd, 0)
	sub.CurrentPeriodEnd = &periodEnd

	err = s.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("stripe subscription created", "workspace_id", sub.WorkspaceID, "plan_id", planID, "stripe_sub_id", subscription.ID)
	return nil
}

func (s *StripeProvider) handleSubscriptionUpdated(data json.RawMessage, providerEventID ...string) error {
	var subscription struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		CurrentPeriodEnd  int64  `json:"current_period_end"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
		Items             struct {
			Data []struct {
				Price struct {
					ID         string `json:"id"`
					UnitAmount int64  `json:"unit_amount"`
					Currency   string `json:"currency"`
					Recurring  struct {
						Interval string `json:"interval"`
					} `json:"recurring"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}

	err := json.Unmarshal(data, &subscription)
	if err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	sub, err := s.subscriptionService.ByProviderSubscriptionID(subscription.ID)
	if err != nil {
		slog.Warn("stripe subscription not found, skipping update", "stripe_sub_id", subscription.ID)
		return nil
	}

	if len(subscription.Items.Data) > 0 {
		priceID := subscription.Items.Data[0].Price.ID
		planID := s.getLocalPlanID(priceID)
		if planID != "" {
			sub.PlanID = planID
		}

		amount := int(subscription.Items.Data[0].Price.UnitAmount)
		sub.Amount = &amount
		sub.Currency = subscription.Items.Data[0].Price.Currency

		interval := s.mapStripeInterval(subscription.Items.Data[0].Price.Recurring.Interval)
		sub.Interval = &interval
	}

	sub.Status = s.mapStripeStatus(subscription.Status)

	// If subscription is set to cancel at period end, mark as cancelled
	if subscription.CancelAtPeriodEnd {
		sub.Status = model.SubscriptionStatusCancelled
	}

	periodEnd := time.Unix(subscription.CurrentPeriodEnd, 0)
	sub.CurrentPeriodEnd = &periodEnd

	err = s.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("stripe subscription updated", "workspace_id", sub.WorkspaceID, "stripe_sub_id", subscription.ID, "status", sub.Status)
	return nil
}

func (s *StripeProvider) handleSubscriptionDeleted(data json.RawMessage, providerEventID ...string) error {
	var subscription struct {
		ID string `json:"id"`
	}

	err := json.Unmarshal(data, &subscription)
	if err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	sub, err := s.subscriptionService.ByProviderSubscriptionID(subscription.ID)
	if err != nil {
		slog.Warn("stripe subscription not found, ignoring deletion", "stripe_sub_id", subscription.ID)
		return nil
	}

	if sub.PlanID == model.SubscriptionPlanFree {
		slog.Warn("stripe subscription already free, ignoring deletion")
		return nil
	}

	err = s.subscriptionService.EndPaidAccess(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to downgrade subscription: %w", err)
	}

	slog.Info("stripe paid access ended", "workspace_id", sub.WorkspaceID, "stripe_sub_id", subscription.ID)
	return nil
}

func (s *StripeProvider) handleInvoicePaymentSucceeded(data json.RawMessage, providerEventID ...string) error {
	var invoice struct {
		SubscriptionID string `json:"subscription"`
	}

	err := json.Unmarshal(data, &invoice)
	if err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	if invoice.SubscriptionID == "" {
		// One-time payment, not subscription-related
		return nil
	}

	sub, err := s.subscriptionService.ByProviderSubscriptionID(invoice.SubscriptionID)
	if err != nil {
		slog.Warn("stripe invoice has unknown subscription, skipping", "subscription_id", invoice.SubscriptionID)
		return nil
	}

	// Ensure subscription is active after successful payment
	if sub.Status != model.SubscriptionStatusActive {
		sub.Status = model.SubscriptionStatusActive
		err = s.subscriptionService.UpdateSubscription(sub, providerEventID...)
		if err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}
	}

	slog.Info("stripe invoice payment succeeded", "workspace_id", sub.WorkspaceID, "subscription_id", invoice.SubscriptionID)
	return nil
}

func (s *StripeProvider) handleInvoicePaymentFailed(data json.RawMessage, providerEventID ...string) error {
	var invoice struct {
		SubscriptionID string `json:"subscription"`
	}

	err := json.Unmarshal(data, &invoice)
	if err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	if invoice.SubscriptionID == "" {
		return nil
	}

	sub, err := s.subscriptionService.ByProviderSubscriptionID(invoice.SubscriptionID)
	if err != nil {
		slog.Warn("stripe invoice has unknown subscription, skipping", "subscription_id", invoice.SubscriptionID)
		return nil
	}

	slog.Warn("stripe invoice payment failed", "workspace_id", sub.WorkspaceID, "subscription_id", invoice.SubscriptionID)
	// Note: Don't automatically cancel. Stripe will retry and eventually send subscription.deleted
	return nil
}

func (s *StripeProvider) getStripePriceID(planID, interval string) string {
	switch {
	case planID == model.SubscriptionPlanPro && interval == model.SubscriptionIntervalMonthly:
		return s.cfg.StripePriceIDProMonthly
	case planID == model.SubscriptionPlanPro && interval == model.SubscriptionIntervalYearly:
		return s.cfg.StripePriceIDProYearly
	case planID == model.SubscriptionPlanEnterprise && interval == model.SubscriptionIntervalMonthly:
		return s.cfg.StripePriceIDEnterpriseMonthly
	case planID == model.SubscriptionPlanEnterprise && interval == model.SubscriptionIntervalYearly:
		return s.cfg.StripePriceIDEnterpriseYearly
	default:
		return ""
	}
}

func (s *StripeProvider) getLocalPlanID(priceID string) string {
	switch priceID {
	case s.cfg.StripePriceIDProMonthly, s.cfg.StripePriceIDProYearly:
		return model.SubscriptionPlanPro
	case s.cfg.StripePriceIDEnterpriseMonthly, s.cfg.StripePriceIDEnterpriseYearly:
		return model.SubscriptionPlanEnterprise
	default:
		return ""
	}
}

func (s *StripeProvider) mapStripeStatus(status string) string {
	switch status {
	case "active":
		return model.SubscriptionStatusActive
	case "trialing":
		return model.SubscriptionStatusTrialing
	case "canceled":
		return model.SubscriptionStatusCancelled
	case "incomplete_expired", "unpaid":
		return model.SubscriptionStatusInactive
	default:
		return status
	}
}

func (s *StripeProvider) mapStripeInterval(interval string) string {
	switch interval {
	case "month":
		return model.SubscriptionIntervalMonthly
	case "year":
		return model.SubscriptionIntervalYearly
	default:
		return interval
	}
}
