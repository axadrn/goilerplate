package payment

import "net/http"

// Provider defines the interface that all payment providers must implement
type Provider interface {
	// CreateCheckoutURL creates a checkout session and returns the URL
	CreateCheckoutURL(userID, planID, interval, customerEmail, customerName string) (string, error)

	// CustomerPortalURL creates a customer portal session and returns the URL
	CustomerPortalURL(userID string) (string, error)

	// HandleWebhook processes webhook events from the payment provider
	HandleWebhook(payload []byte, headers http.Header) error

	// Name returns the configured provider name.
	Name() string
}
