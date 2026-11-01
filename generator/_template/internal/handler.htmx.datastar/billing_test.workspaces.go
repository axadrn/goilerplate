package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

func TestCreateCheckoutUsesAuthenticatedUserAndDefaultInterval(t *testing.T) {
	provider := &paymentProviderStub{checkoutURL: "https://payments.example.com/checkout"}
	handler := NewBillingHandler(provider, nil)
	form := url.Values{"plan_id": {model.SubscriptionPlanPro}}
	request := authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", form)
	recorder := httptest.NewRecorder()

	handler.CreateCheckout(recorder, request)

	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != provider.checkoutURL {
		t.Fatalf("status = %d, location = %q", recorder.Code, recorder.Header().Get("Location"))
	}
	if provider.checkoutRequest.workspaceID != "workspace-1" || provider.checkoutRequest.workspaceSlug != "test-workspace" || provider.checkoutRequest.planID != model.SubscriptionPlanPro {
		t.Fatalf("checkout request = %#v", provider.checkoutRequest)
	}
	if provider.checkoutRequest.interval != model.SubscriptionIntervalMonthly {
		t.Fatalf("checkout interval = %q, want monthly", provider.checkoutRequest.interval)
	}
	if provider.checkoutRequest.email != "user@example.com" || provider.checkoutRequest.name != "Test Workspace" {
		t.Fatalf("checkout identity = %#v", provider.checkoutRequest)
	}
}

func TestCreateCheckoutRejectsInvalidRequestAndProviderFailure(t *testing.T) {
	t.Run("missing plan", func(t *testing.T) {
		provider := &paymentProviderStub{checkoutURL: "https://payments.example.com/checkout"}
		handler := NewBillingHandler(provider, nil)
		recorder := httptest.NewRecorder()

		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", nil))

		if recorder.Code != http.StatusBadRequest || provider.checkoutCalls != 0 {
			t.Fatalf("status = %d, checkout calls = %d", recorder.Code, provider.checkoutCalls)
		}
	})

	t.Run("unknown plan", func(t *testing.T) {
		provider := &paymentProviderStub{}
		handler := NewBillingHandler(provider, nil)
		recorder := httptest.NewRecorder()
		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", url.Values{"plan_id": {"invented"}}))
		if recorder.Code != http.StatusBadRequest || provider.checkoutCalls != 0 {
			t.Fatalf("status = %d, checkout calls = %d", recorder.Code, provider.checkoutCalls)
		}
	})

	t.Run("invalid interval", func(t *testing.T) {
		provider := &paymentProviderStub{}
		handler := NewBillingHandler(provider, nil)
		recorder := httptest.NewRecorder()
		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", url.Values{"plan_id": {model.SubscriptionPlanPro}, "interval": {"weekly"}}))
		if recorder.Code != http.StatusBadRequest || provider.checkoutCalls != 0 {
			t.Fatalf("status = %d, checkout calls = %d", recorder.Code, provider.checkoutCalls)
		}
	})

	t.Run("provider failure", func(t *testing.T) {
		provider := &paymentProviderStub{checkoutErr: errors.New("provider unavailable")}
		handler := NewBillingHandler(provider, nil)
		form := url.Values{"plan_id": {model.SubscriptionPlanPro}, "interval": {model.SubscriptionIntervalYearly}}
		recorder := httptest.NewRecorder()

		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", form))

		if recorder.Code != http.StatusInternalServerError || provider.checkoutRequest.interval != model.SubscriptionIntervalYearly {
			t.Fatalf("status = %d, checkout request = %#v", recorder.Code, provider.checkoutRequest)
		}
	})

	t.Run("checkout already pending", func(t *testing.T) {
		provider := &paymentProviderStub{checkoutErr: repository.ErrCheckoutPending}
		handler := NewBillingHandler(provider, nil)
		recorder := httptest.NewRecorder()

		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", url.Values{"plan_id": {model.SubscriptionPlanPro}}))

		if recorder.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
		}
	})

	t.Run("different checkout already open", func(t *testing.T) {
		provider := &paymentProviderStub{checkoutErr: repository.ErrCheckoutSelection}
		handler := NewBillingHandler(provider, nil)
		recorder := httptest.NewRecorder()

		handler.CreateCheckout(recorder, authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", url.Values{"plan_id": {model.SubscriptionPlanEnterprise}}))

		if recorder.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
		}
	})
}

func TestBillingRequiresOwner(t *testing.T) {
	provider := &paymentProviderStub{}
	handler := NewBillingHandler(provider, nil)
	request := authenticatedBillingRequest(http.MethodPost, "/app/test-workspace/billing/checkout", url.Values{"plan_id": {model.SubscriptionPlanPro}})
	request = request.WithContext(ctxkeys.WithWorkspaceMember(request.Context(), &model.WorkspaceMember{WorkspaceID: "workspace-1", UserID: "user-1", Role: model.WorkspaceRoleAdmin}))
	recorder := httptest.NewRecorder()

	handler.CreateCheckout(recorder, request)
	if recorder.Code != http.StatusForbidden || provider.checkoutCalls != 0 {
		t.Fatalf("status = %d, checkout calls = %d", recorder.Code, provider.checkoutCalls)
	}
}

func TestBillingWebhookDelegatesPayloadAndHeaders(t *testing.T) {
	provider := &paymentProviderStub{}
	handler := NewBillingHandler(provider, nil)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(`{"type":"subscription.updated"}`))
	request.Header.Set("Webhook-Signature", "signature")
	recorder := httptest.NewRecorder()

	handler.Webhook(recorder, request)

	if recorder.Code != http.StatusOK || provider.webhookCalls != 1 {
		t.Fatalf("status = %d, webhook calls = %d", recorder.Code, provider.webhookCalls)
	}
	if string(provider.webhookPayload) != `{"type":"subscription.updated"}` || provider.webhookHeaders.Get("Webhook-Signature") != "signature" {
		t.Fatalf("webhook payload = %q, headers = %#v", provider.webhookPayload, provider.webhookHeaders)
	}
}

func TestBillingWebhookReturnsBadRequestForProviderError(t *testing.T) {
	provider := &paymentProviderStub{webhookErr: errors.New("invalid signature")}
	handler := NewBillingHandler(provider, nil)
	recorder := httptest.NewRecorder()

	handler.Webhook(recorder, httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader("{}")))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestBillingWebhookRejectsPayloadLargerThanOneMegabyte(t *testing.T) {
	provider := &paymentProviderStub{}
	handler := NewBillingHandler(provider, nil)
	recorder := httptest.NewRecorder()

	handler.Webhook(recorder, httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(strings.Repeat("x", (1<<20)+1))))

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	if provider.webhookCalls != 0 {
		t.Fatalf("webhook calls = %d, want 0", provider.webhookCalls)
	}
}

func TestBillingManagementRedirectsToProvider(t *testing.T) {
	provider := &paymentProviderStub{portalURL: "https://payments.example.com/portal"}
	handler := NewBillingHandler(provider, nil)
	recorder := httptest.NewRecorder()

	handler.BillingManagement(recorder, authenticatedBillingRequest(http.MethodGet, "/app/test-workspace/billing/manage", nil))

	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != provider.portalURL {
		t.Fatalf("status = %d, location = %q", recorder.Code, recorder.Header().Get("Location"))
	}
	if provider.portalWorkspaceID != "workspace-1" || provider.portalWorkspaceSlug != "test-workspace" {
		t.Fatalf("portal workspace = %q, slug = %q", provider.portalWorkspaceID, provider.portalWorkspaceSlug)
	}
}

func authenticatedBillingRequest(method, target string, form url.Values) *http.Request {
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := ctxkeys.WithUser(request.Context(), &model.User{ID: "user-1", Email: "user@example.com"})
	ctx = ctxkeys.WithWorkspace(ctx, &model.Workspace{ID: "workspace-1", Name: "Test Workspace", Slug: "test-workspace"})
	ctx = ctxkeys.WithWorkspaceMember(ctx, &model.WorkspaceMember{WorkspaceID: "workspace-1", UserID: "user-1", Role: model.WorkspaceRoleOwner})
	return request.WithContext(ctx)
}

type paymentProviderStub struct {
	checkoutURL         string
	checkoutErr         error
	checkoutCalls       int
	checkoutRequest     checkoutRequest
	portalURL           string
	portalErr           error
	portalWorkspaceID   string
	portalWorkspaceSlug string
	webhookPayload      []byte
	webhookHeaders      http.Header
	webhookErr          error
	webhookCalls        int
}

type checkoutRequest struct {
	workspaceID   string
	workspaceSlug string
	planID        string
	interval      string
	email         string
	name          string
}

func (p *paymentProviderStub) CreateCheckoutURL(workspaceID, workspaceSlug, planID, interval, customerEmail, customerName string) (string, error) {
	p.checkoutCalls++
	p.checkoutRequest = checkoutRequest{
		workspaceID:   workspaceID,
		workspaceSlug: workspaceSlug,
		planID:        planID,
		interval:      interval,
		email:         customerEmail,
		name:          customerName,
	}
	return p.checkoutURL, p.checkoutErr
}

func (p *paymentProviderStub) BillingManagementURL(workspaceID, workspaceSlug string) (string, error) {
	p.portalWorkspaceID = workspaceID
	p.portalWorkspaceSlug = workspaceSlug
	return p.portalURL, p.portalErr
}

func (p *paymentProviderStub) HandleWebhook(payload []byte, headers http.Header) error {
	p.webhookCalls++
	p.webhookPayload = append([]byte(nil), payload...)
	p.webhookHeaders = headers.Clone()
	return p.webhookErr
}

func (p *paymentProviderStub) Name() string {
	return "stub"
}
