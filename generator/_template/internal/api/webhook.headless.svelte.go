package api

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service/payment"
)

// maxWebhookBytes caps a webhook payload. Provider events stay far below it.
const maxWebhookBytes = 1 << 20

type webhookResponse struct {
	Received bool `json:"received"`
}

// PaymentWebhook receives the payment provider webhooks at /webhooks/payment.
// The provider verifies the signature in HandleWebhook. Payloads over 1 MiB
// answer 413.
func PaymentWebhook(provider payment.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBytes))
		if err != nil {
			slog.Error("failed to read webhook payload", "error", err)
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				Error(w, http.StatusRequestEntityTooLarge, "body_too_large", "The webhook payload is too large.")
				return
			}
			Error(w, http.StatusBadRequest, "invalid_body", "The webhook payload could not be read.")
			return
		}
		err = provider.HandleWebhook(payload, r.Header)
		if err != nil {
			slog.Error("failed to handle webhook", "error", err, "provider", provider.Name())
			Error(w, http.StatusBadRequest, "invalid_webhook", "The webhook could not be processed.")
			return
		}
		JSON(w, http.StatusOK, webhookResponse{Received: true})
	}
}
