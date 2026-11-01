package api

import (
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service/payment"
)

// BillingHandler serves the subscription of the signed-in user, checkout,
// and the provider customer portal.
type BillingHandler struct {
	paymentService payment.Provider
}

func NewBillingHandler(paymentService payment.Provider) *BillingHandler {
	return &BillingHandler{paymentService: paymentService}
}

// Billing returns the subscription, its access, and the purchasable plans.
func (h *BillingHandler) Billing(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, NewBilling(ctxkeys.Subscription(r.Context())))
}

// CreateCheckout starts a provider checkout and returns its URL.
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	var request checkoutRequest
	if !Decode(w, r, &request) {
		return
	}
	if !checkoutSelection(w, &request) {
		return
	}

	checkoutURL, err := h.paymentService.CreateCheckoutURL(user.ID, request.PlanID, request.Interval, user.Email, user.Name)
	if err != nil {
		slog.Error("failed to create checkout", "error", err, "user_id", user.ID, "plan_id", request.PlanID, "provider", h.paymentService.Name())
		Error(w, http.StatusInternalServerError, "internal", "Checkout could not be started. Please try again.")
		return
	}

	slog.Info("checkout created", "user_id", user.ID, "provider", h.paymentService.Name())
	JSON(w, http.StatusOK, URLResponse{URL: checkoutURL})
}

// CustomerPortal returns the URL of the provider customer portal. A user
// without a provider customer gets 409 no_billing_account.
func (h *BillingHandler) CustomerPortal(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if !hasBillingAccount(ctxkeys.Subscription(r.Context())) {
		noBillingAccount(w)
		return
	}

	portalURL, err := h.paymentService.CustomerPortalURL(user.ID)
	if err != nil {
		slog.Error("failed to get customer portal", "error", err, "user_id", user.ID, "provider", h.paymentService.Name())
		Error(w, http.StatusInternalServerError, "internal", "The customer portal could not be opened. Please try again.")
		return
	}
	JSON(w, http.StatusOK, URLResponse{URL: portalURL})
}
