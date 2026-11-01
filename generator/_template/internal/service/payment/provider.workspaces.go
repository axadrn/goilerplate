package payment

import (
	"errors"
	"net/http"
	"time"
)

var (
	errProviderRejected = errors.New("payment provider rejected the request")
)

const checkoutWebhookGrace = 4 * 24 * time.Hour

// Provider defines the interface that all payment providers must implement
type Provider interface {
	// CreateCheckoutURL creates a checkout session and returns the URL
	CreateCheckoutURL(workspaceID, workspaceSlug, planID, interval, customerEmail, customerName string) (string, error)

	// BillingManagementURL creates a provider-hosted, workspace-safe billing URL.
	BillingManagementURL(workspaceID, workspaceSlug string) (string, error)

	// HandleWebhook processes webhook events from the payment provider
	HandleWebhook(payload []byte, headers http.Header) error

	// Name returns the configured provider name.
	Name() string
}
