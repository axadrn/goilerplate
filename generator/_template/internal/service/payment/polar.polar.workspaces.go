package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/apierrors"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/service"
)

type PolarProvider struct {
	cfg                 *config.Config
	subscriptionService *service.SubscriptionService
	client              *polargo.Polar
}

func NewPolarProvider(cfg *config.Config, subscriptionService *service.SubscriptionService) *PolarProvider {
	var serverOption polargo.SDKOption
	if cfg.PolarSandboxMode {
		serverOption = polargo.WithServer(polargo.ServerSandbox)
		slog.Info("polar using sandbox mode", "app_env", cfg.AppEnv)
	} else {
		serverOption = polargo.WithServer(polargo.ServerProduction)
		slog.Info("polar using production mode", "app_env", cfg.AppEnv)
	}

	client := polargo.New(
		polargo.WithSecurity(cfg.PolarAPIKey),
		serverOption,
	)

	return &PolarProvider{
		cfg:                 cfg,
		subscriptionService: subscriptionService,
		client:              client,
	}
}

func (p *PolarProvider) Name() string {
	return model.ProviderPolar
}

func (p *PolarProvider) CreateCheckoutURL(workspaceID, workspaceSlug, planID, interval, customerEmail, customerName string) (string, error) {
	ctx := context.Background()

	sub, err := p.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	productID := p.getPolarProductID(planID, interval)
	if productID == "" {
		return "", fmt.Errorf("no product configured for plan: %s (%s)", planID, interval)
	}
	checkoutExpiresAt := time.Now().Add(48 * time.Hour)
	checkoutURL, attemptID, reused, err := p.subscriptionService.BeginCheckout(workspaceID, p.Name(), planID, interval, checkoutExpiresAt, checkoutExpiresAt.Add(checkoutWebhookGrace))
	if err != nil {
		return "", err
	}
	if reused {
		return checkoutURL, nil
	}

	successURL := fmt.Sprintf("%s/app/%s/billing", p.cfg.AppURL, workspaceSlug)
	returnURL := fmt.Sprintf("%s/app/%s/billing", p.cfg.AppURL, workspaceSlug)

	metadata := map[string]components.CheckoutCreateMetadata{
		"workspace_id":        components.CreateCheckoutCreateMetadataStr(workspaceID),
		"subscription_id":     components.CreateCheckoutCreateMetadataStr(sub.ID),
		"plan_id":             components.CreateCheckoutCreateMetadataStr(planID),
		"checkout_attempt_id": components.CreateCheckoutCreateMetadataStr(attemptID),
	}

	res, err := p.client.Checkouts.Create(ctx, components.CheckoutCreate{
		Products:           []string{productID},
		SuccessURL:         polargo.String(successURL),
		ReturnURL:          polargo.String(returnURL),
		ExternalCustomerID: polargo.String(workspaceID),
		CustomerEmail:      polargo.String(customerEmail),
		CustomerName:       polargo.String(customerName),
		AllowDiscountCodes: polargo.Bool(true),
		Metadata:           metadata,
	})

	if err != nil {
		var apiError *apierrors.APIError
		var validationError *apierrors.HTTPValidationError
		if (errors.As(err, &apiError) && apiError.StatusCode >= 400 && apiError.StatusCode < 500) || errors.As(err, &validationError) {
			p.subscriptionService.RejectCheckout(workspaceID, attemptID)
		}
		return "", fmt.Errorf("failed to create checkout: %w", err)
	}

	if res == nil || res.Checkout == nil {
		return "", fmt.Errorf("checkout response is nil")
	}
	if res.Checkout.URL == "" {
		return "", fmt.Errorf("checkout response has no URL")
	}
	if !res.Checkout.ExpiresAt.IsZero() {
		checkoutExpiresAt = res.Checkout.ExpiresAt
	}
	if err := p.subscriptionService.FinishCheckout(workspaceID, attemptID, res.Checkout.URL, checkoutExpiresAt, checkoutExpiresAt.Add(checkoutWebhookGrace)); err != nil {
		return "", err
	}
	slog.Info("polar checkout created", "workspace_id", workspaceID, "plan_id", planID, "checkout_id", res.Checkout.ID)
	return res.Checkout.URL, nil
}

func (p *PolarProvider) BillingManagementURL(workspaceID, workspaceSlug string) (string, error) {
	ctx := context.Background()

	sub, err := p.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.ProviderCustomerID == nil || *sub.ProviderCustomerID == "" {
		return "", fmt.Errorf("no customer portal available for free subscriptions")
	}

	returnURL := fmt.Sprintf("%s/app/%s/billing", p.cfg.AppURL, workspaceSlug)

	sessionCreate := operations.CreateCustomerSessionsCreateCustomerSessionCreateCustomerSessionCustomerIDCreate(
		components.CustomerSessionCustomerIDCreate{
			CustomerID: *sub.ProviderCustomerID,
			ReturnURL:  polargo.String(returnURL),
		},
	)
	res, err := p.client.CustomerSessions.Create(ctx, sessionCreate)

	if err != nil {
		return "", fmt.Errorf("failed to create customer portal session: %w", err)
	}

	if res == nil || res.CustomerSession == nil {
		return "", fmt.Errorf("customer portal response is nil")
	}

	slog.Info("polar customer portal session created", "workspace_id", workspaceID)
	return res.CustomerSession.CustomerPortalURL, nil
}

func (p *PolarProvider) HandleWebhook(payload []byte, headers http.Header) error {
	webhookID := headers.Get("webhook-id")
	timestamp := headers.Get("webhook-timestamp")
	signature := headers.Get("webhook-signature")

	if p.cfg.PolarWebhookSecret == "" {
		slog.Warn("polar no webhook secret configured, skipping signature verification")
	} else {
		wh, err := standardwebhooks.NewWebhookRaw([]byte(p.cfg.PolarWebhookSecret))
		if err != nil {
			return fmt.Errorf("failed to create webhook verifier: %w", err)
		}

		httpHeaders := http.Header{}
		httpHeaders.Set("webhook-id", webhookID)
		httpHeaders.Set("webhook-timestamp", timestamp)
		httpHeaders.Set("webhook-signature", signature)

		err = wh.Verify(payload, httpHeaders)
		if err != nil {
			return fmt.Errorf("invalid webhook signature: %w", err)
		}
	}

	var event struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	err := json.Unmarshal(payload, &event)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	slog.Info("polar webhook received", "event_type", event.Type)

	switch event.Type {
	case "subscription.created":
		return p.handleSubscriptionCreated(event.Data, webhookID)
	case "subscription.updated":
		return p.handleSubscriptionUpdated(event.Data, webhookID)
	case "subscription.canceled":
		return p.handleSubscriptionCanceled(event.Data, webhookID)
	case "subscription.uncanceled":
		return p.handleSubscriptionUncanceled(event.Data, webhookID)
	case "subscription.revoked":
		return p.handleSubscriptionRevoked(event.Data, webhookID)
	default:
		slog.Warn("polar webhook unknown event type", "event_type", event.Type)
		return nil
	}
}

func (p *PolarProvider) handleSubscriptionCreated(data json.RawMessage, providerEventID ...string) error {
	var subscription struct {
		ID                string            `json:"id"`
		CustomerID        string            `json:"customer_id"`
		Amount            *int              `json:"amount"`
		Currency          *string           `json:"currency"`
		RecurringInterval *string           `json:"recurring_interval"`
		Status            string            `json:"status"`
		CurrentPeriodEnd  *string           `json:"current_period_end"`
		Metadata          map[string]string `json:"metadata"`
	}

	err := json.Unmarshal(data, &subscription)
	if err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	workspaceID := subscription.Metadata["workspace_id"]
	planID := subscription.Metadata["plan_id"]

	if workspaceID == "" {
		slog.Warn("polar webhook has no workspace_id in subscription metadata, skipping")
		return nil
	}

	sub, err := p.subscriptionService.Subscription(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if planID != "" {
		sub.PlanID = planID
	}

	sub.Provider = model.ProviderPolar
	sub.ProviderCustomerID = &subscription.CustomerID
	sub.ProviderSubscriptionID = &subscription.ID
	sub.ClearCheckout()
	sub.Status = p.mapPolarStatus(subscription.Status)

	if subscription.Amount != nil {
		sub.Amount = subscription.Amount
	}

	if subscription.Currency != nil {
		sub.Currency = *subscription.Currency
	}

	if subscription.RecurringInterval != nil {
		sub.Interval = subscription.RecurringInterval
	}

	if subscription.CurrentPeriodEnd != nil {
		periodEnd, err := parseTime(*subscription.CurrentPeriodEnd)
		if err == nil {
			sub.CurrentPeriodEnd = &periodEnd
		}
	}

	err = p.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription created", "workspace_id", workspaceID, "plan_id", planID, "polar_sub_id", subscription.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionUpdated(data json.RawMessage, providerEventID ...string) error {
	var subscription struct {
		ID                string  `json:"id"`
		Amount            *int    `json:"amount"`
		Currency          *string `json:"currency"`
		RecurringInterval *string `json:"recurring_interval"`
		Status            string  `json:"status"`
		CurrentPeriodEnd  *string `json:"current_period_end"`
		EndedAt           *string `json:"ended_at"`
		ProductID         *string `json:"product_id"`
	}

	err := json.Unmarshal(data, &subscription)
	if err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := p.subscriptionService.ByProviderSubscriptionID(subscription.ID)
	if err != nil {
		slog.Warn("polar subscription not found, skipping update", "polar_sub_id", subscription.ID)
		return nil
	}

	if subscription.EndedAt != nil {
		err = p.subscriptionService.EndPaidAccess(sub, providerEventID...)
		if err != nil {
			return fmt.Errorf("failed to downgrade subscription: %w", err)
		}
		slog.Info("polar paid access ended", "workspace_id", sub.WorkspaceID, "polar_sub_id", subscription.ID)
		return nil
	}

	if subscription.ProductID != nil {
		planID := p.getLocalPlanID(*subscription.ProductID)
		if planID != "" {
			sub.PlanID = planID
		}
	}

	providerSubID := subscription.ID
	sub.ProviderSubscriptionID = &providerSubID

	if subscription.Amount != nil {
		sub.Amount = subscription.Amount
	}

	if subscription.Currency != nil {
		sub.Currency = *subscription.Currency
	}

	if subscription.RecurringInterval != nil {
		sub.Interval = subscription.RecurringInterval
	}

	if subscription.CurrentPeriodEnd != nil {
		periodEnd, err := parseTime(*subscription.CurrentPeriodEnd)
		if err == nil {
			sub.CurrentPeriodEnd = &periodEnd
		}
	}

	if subscription.Status != "" {
		sub.Status = p.mapPolarStatus(subscription.Status)
	}

	err = p.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription updated", "workspace_id", sub.WorkspaceID, "polar_sub_id", subscription.ID)
	return nil
}

func (p *PolarProvider) mapPolarStatus(status string) string {
	switch status {
	case "active":
		return model.SubscriptionStatusActive
	case "trialing":
		return model.SubscriptionStatusTrialing
	case "past_due":
		return model.SubscriptionStatusPastDue
	case "incomplete":
		return model.SubscriptionStatusIncomplete
	case "canceled", "cancelled":
		return model.SubscriptionStatusCancelled
	case "incomplete_expired", "revoked", "unpaid":
		return model.SubscriptionStatusInactive
	default:
		return status
	}
}

func (p *PolarProvider) handleSubscriptionCanceled(data json.RawMessage, providerEventID ...string) error {
	var subData struct {
		ID               string  `json:"id"`
		CurrentPeriodEnd *string `json:"current_period_end"`
	}

	err := json.Unmarshal(data, &subData)
	if err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := p.subscriptionService.ByProviderSubscriptionID(subData.ID)
	if err != nil {
		slog.Warn("polar subscription not found, ignoring cancellation", "polar_sub_id", subData.ID)
		return nil
	}

	if sub.PlanID == model.SubscriptionPlanFree {
		slog.Warn("polar subscription already free, ignoring cancellation")
		return nil
	}

	sub.Status = model.SubscriptionStatusCancelled

	if subData.CurrentPeriodEnd != nil {
		periodEnd, err := parseTime(*subData.CurrentPeriodEnd)
		if err == nil {
			sub.CurrentPeriodEnd = &periodEnd
		}
	}

	err = p.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription canceled", "workspace_id", sub.WorkspaceID, "polar_sub_id", subData.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionUncanceled(data json.RawMessage, providerEventID ...string) error {
	var subData struct {
		ID string `json:"id"`
	}

	err := json.Unmarshal(data, &subData)
	if err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := p.subscriptionService.ByProviderSubscriptionID(subData.ID)
	if err != nil {
		slog.Warn("polar subscription not found, ignoring uncanceled event", "polar_sub_id", subData.ID)
		return nil
	}

	sub.Status = model.SubscriptionStatusActive

	err = p.subscriptionService.UpdateSubscription(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription uncanceled", "workspace_id", sub.WorkspaceID, "polar_sub_id", subData.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionRevoked(data json.RawMessage, providerEventID ...string) error {
	var subData struct {
		ID string `json:"id"`
	}

	err := json.Unmarshal(data, &subData)
	if err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := p.subscriptionService.ByProviderSubscriptionID(subData.ID)
	if err != nil {
		slog.Warn("polar subscription not found, ignoring revoked event", "polar_sub_id", subData.ID)
		return nil
	}

	if sub.PlanID == model.SubscriptionPlanFree {
		slog.Warn("polar subscription already free, ignoring revoked event")
		return nil
	}

	err = p.subscriptionService.EndPaidAccess(sub, providerEventID...)
	if err != nil {
		return fmt.Errorf("failed to downgrade subscription: %w", err)
	}

	slog.Info("polar paid access revoked", "workspace_id", sub.WorkspaceID, "polar_sub_id", subData.ID)
	return nil
}

func (p *PolarProvider) getPolarProductID(planID, interval string) string {
	switch {
	case planID == model.SubscriptionPlanPro && interval == model.SubscriptionIntervalMonthly:
		return p.cfg.PolarProductIDProMonthly
	case planID == model.SubscriptionPlanPro && interval == model.SubscriptionIntervalYearly:
		return p.cfg.PolarProductIDProYearly
	case planID == model.SubscriptionPlanEnterprise && interval == model.SubscriptionIntervalMonthly:
		return p.cfg.PolarProductIDEnterpriseMonthly
	case planID == model.SubscriptionPlanEnterprise && interval == model.SubscriptionIntervalYearly:
		return p.cfg.PolarProductIDEnterpriseYearly
	default:
		return ""
	}
}

func (p *PolarProvider) getLocalPlanID(productID string) string {
	switch productID {
	case p.cfg.PolarProductIDProMonthly, p.cfg.PolarProductIDProYearly:
		return model.SubscriptionPlanPro
	case p.cfg.PolarProductIDEnterpriseMonthly, p.cfg.PolarProductIDEnterpriseYearly:
		return model.SubscriptionPlanEnterprise
	default:
		return ""
	}
}
