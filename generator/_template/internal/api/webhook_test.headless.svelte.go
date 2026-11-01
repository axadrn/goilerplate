package api_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/api"
)

func TestPaymentWebhookPassesThePayloadToTheProvider(t *testing.T) {
	payments := &paymentStub{name: "test"}
	recorder := httptest.NewRecorder()

	api.PaymentWebhook(payments)(recorder, httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(`{"type":"checkout"}`)))

	if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != `{"received":true}` {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if len(payments.webhooks) != 1 || payments.webhooks[0] != `{"type":"checkout"}` {
		t.Fatalf("webhooks = %v", payments.webhooks)
	}
}

func TestPaymentWebhookRejectsWhatTheProviderRejects(t *testing.T) {
	payments := &paymentStub{name: "test", webhookErr: errors.New("bad signature")}
	recorder := httptest.NewRecorder()

	api.PaymentWebhook(payments)(recorder, httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(`{}`)))

	expectError(t, recorder, http.StatusBadRequest, "invalid_webhook")
}

func TestPaymentWebhookRejectsPayloadLargerThanOneMegabyte(t *testing.T) {
	payments := &paymentStub{name: "test"}
	recorder := httptest.NewRecorder()

	api.PaymentWebhook(payments)(recorder, httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(strings.Repeat("x", (1<<20)+1))))

	expectError(t, recorder, http.StatusRequestEntityTooLarge, "body_too_large")
	if len(payments.webhooks) != 0 {
		t.Fatalf("webhooks = %d, want 0", len(payments.webhooks))
	}
}
