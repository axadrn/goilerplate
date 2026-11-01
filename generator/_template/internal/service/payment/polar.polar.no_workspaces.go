package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	polargo "github.com/polarsource/polar-go"
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

func (p *PolarProvider) CreateCheckoutURL(userID, planID, interval, customerEmail, customerName string) (string, error) {
	ctx := context.Background()

	sub, err := p.subscriptionService.Subscription(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	productID := p.getPolarProductID(planID, interval)
	if productID == "" {
		return "", fmt.Errorf("no product configured for plan: %s (%s)", planID, interval)
	}

	successURL := fmt.Sprintf("%s/app/billing", p.cfg.AppURL)
	returnURL := fmt.Sprintf("%s/app/billing", p.cfg.AppURL)

	metadata := map[string]components.CheckoutCreateMetadata{
		"user_id":         components.CreateCheckoutCreateMetadataStr(userID),
		"subscription_id": components.CreateCheckoutCreateMetadataStr(sub.ID),
		"plan_id":         components.CreateCheckoutCreateMetadataStr(planID),
	}

	res, err := p.client.Checkouts.Create(ctx, components.CheckoutCreate{
		Products:           []string{productID},
		SuccessURL:         polargo.String(successURL),
		ReturnURL:          polargo.String(returnURL),
		CustomerEmail:      polargo.String(customerEmail),
		CustomerName:       polargo.String(customerName),
		AllowDiscountCodes: polargo.Bool(true),
		Metadata:           metadata,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create checkout: %w", err)
	}

	if res == nil || res.Checkout == nil {
		return "", fmt.Errorf("checkout response is nil")
	}

	slog.Info("polar checkout created", "user_id", userID, "plan_id", planID, "checkout_id", res.Checkout.ID)
	return res.Checkout.URL, nil
}

func (p *PolarProvider) CustomerPortalURL(userID string) (string, error) {
	ctx := context.Background()

	sub, err := p.subscriptionService.Subscription(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.ProviderCustomerID == nil || *sub.ProviderCustomerID == "" {
		return "", fmt.Errorf("no customer portal available for free subscriptions")
	}

	returnURL := fmt.Sprintf("%s/app/billing", p.cfg.AppURL)

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

	slog.Info("polar customer portal session created", "user_id", userID)
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
		return p.handleSubscriptionCreated(event.Data)
	case "subscription.updated":
		return p.handleSubscriptionUpdated(event.Data)
	case "subscription.canceled":
		return p.handleSubscriptionCanceled(event.Data)
	case "subscription.uncanceled":
		return p.handleSubscriptionUncanceled(event.Data)
	case "subscription.revoked":
		return p.handleSubscriptionRevoked(event.Data)
	default:
		slog.Warn("polar webhook unknown event type", "event_type", event.Type)
		return nil
	}
}

func (p *PolarProvider) handleSubscriptionCreated(data json.RawMessage) error {
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

	userID := subscription.Metadata["user_id"]
	planID := subscription.Metadata["plan_id"]

	if userID == "" {
		slog.Warn("polar webhook no user_id in subscription metadata, skipping")
		return nil
	}

	sub, err := p.subscriptionService.Subscription(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if planID != "" {
		sub.PlanID = planID
	}

	sub.Provider = model.ProviderPolar
	sub.ProviderCustomerID = &subscription.CustomerID
	sub.ProviderSubscriptionID = &subscription.ID
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

	err = p.subscriptionService.UpdateSubscription(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription created", "user_id", userID, "plan_id", planID, "polar_sub_id", subscription.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionUpdated(data json.RawMessage) error {
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
		err = p.subscriptionService.EndPaidAccess(sub)
		if err != nil {
			return fmt.Errorf("failed to downgrade subscription: %w", err)
		}
		slog.Info("polar paid access ended", "user_id", sub.UserID, "polar_sub_id", subscription.ID)
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

	err = p.subscriptionService.UpdateSubscription(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription updated", "user_id", sub.UserID, "polar_sub_id", subscription.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionCanceled(data json.RawMessage) error {
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

	err = p.subscriptionService.UpdateSubscription(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription canceled", "user_id", sub.UserID, "polar_sub_id", subData.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionUncanceled(data json.RawMessage) error {
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

	err = p.subscriptionService.UpdateSubscription(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	slog.Info("polar subscription uncanceled", "user_id", sub.UserID, "polar_sub_id", subData.ID)
	return nil
}

func (p *PolarProvider) handleSubscriptionRevoked(data json.RawMessage) error {
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

	err = p.subscriptionService.EndPaidAccess(sub)
	if err != nil {
		return fmt.Errorf("failed to downgrade subscription: %w", err)
	}

	slog.Info("polar paid access revoked", "user_id", sub.UserID, "polar_sub_id", subData.ID)
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
