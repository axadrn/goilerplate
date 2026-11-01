package api_test

import (
	"net/http"
	"sync"
)

type checkoutCall struct {
	workspaceID   string
	workspaceSlug string
	planID        string
	interval      string
	email         string
	name          string
}

// paymentStub is the payment test double of the API tests. Name returns the
// provider of the project, so the same tests cover Stripe and Polar.
type paymentStub struct {
	mu            sync.Mutex
	name          string
	checkoutURL   string
	checkoutErr   error
	managementURL string
	managementErr error
	checkouts     []checkoutCall
	portalCalls   []string
	webhookErr    error
	webhooks      []string
}

func (p *paymentStub) CreateCheckoutURL(workspaceID, workspaceSlug, planID, interval, customerEmail, customerName string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkouts = append(p.checkouts, checkoutCall{workspaceID: workspaceID, workspaceSlug: workspaceSlug, planID: planID, interval: interval, email: customerEmail, name: customerName})
	return p.checkoutURL, p.checkoutErr
}

func (p *paymentStub) BillingManagementURL(workspaceID, _ string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.portalCalls = append(p.portalCalls, workspaceID)
	return p.managementURL, p.managementErr
}

func (p *paymentStub) HandleWebhook(payload []byte, _ http.Header) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.webhooks = append(p.webhooks, string(payload))
	return p.webhookErr
}

func (p *paymentStub) Name() string {
	return p.name
}

func (p *paymentStub) checkoutCalls() []checkoutCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]checkoutCall(nil), p.checkouts...)
}
