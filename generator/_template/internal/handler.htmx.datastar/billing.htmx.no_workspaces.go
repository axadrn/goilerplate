package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/service/payment"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
)

type BillingHandler struct {
	subscriptionService *service.SubscriptionService
	paymentService      payment.Provider
}

func NewBillingHandler(subscriptionService *service.SubscriptionService, paymentService payment.Provider) *BillingHandler {
	return &BillingHandler{
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
	}
}

func (h *BillingHandler) BillingPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.Billing())
}

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	planID := r.FormValue("plan_id")
	plan, exists := access.PlanByID(planID)
	if !exists || !plan.Purchasable {
		if isAction(r) {
			renderOOB(w, r, toast.Toast(toast.Props{Title: "Invalid plan", Description: "Choose an available paid plan.", Type: toast.TypeError}), "beforeend:#toast-container")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			render(w, r, pages.ErrorPage(http.StatusBadRequest, "Invalid plan", "Choose an available paid plan."))
		}
		return
	}

	interval := r.FormValue("interval")
	if interval == "" {
		interval = "monthly"
	}

	checkoutURL, err := h.paymentService.CreateCheckoutURL(user.ID, planID, interval, user.Email, user.Name)
	if err != nil {
		slog.Error("failed to create checkout", "error", err, "user_id", user.ID, "plan_id", planID, "provider", h.paymentService.Name())
		if isAction(r) {
			renderOOB(w, r, toast.Toast(toast.Props{Title: "Checkout unavailable", Description: "Checkout could not be started. Please try again.", Type: toast.TypeError}), "beforeend:#toast-container")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Checkout unavailable", "Checkout could not be started. Please try again."))
		}
		return
	}

	slog.Info("redirecting to checkout", "user_id", user.ID, "provider", h.paymentService.Name(), "checkout_url", checkoutURL)
	redirect(w, r, checkoutURL, http.StatusSeeOther)
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read webhook payload", "error", err)
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "Payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}
	defer func() {
		closeErr := r.Body.Close()
		if closeErr != nil {
			slog.Error("failed to close request body", "error", closeErr)
		}
	}()

	err = h.paymentService.HandleWebhook(payload, r.Header)
	if err != nil {
		slog.Error("failed to handle webhook", "error", err, "provider", h.paymentService.Name())
		http.Error(w, "Failed to process webhook", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received": true}`))
}

func (h *BillingHandler) CustomerPortal(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	portalURL, err := h.paymentService.CustomerPortalURL(user.ID)
	if err != nil {
		slog.Error("failed to get customer portal", "error", err, "user_id", user.ID, "provider", h.paymentService.Name())
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Billing unavailable", "The customer portal could not be opened. Please try again."))
		return
	}

	redirect(w, r, portalURL, http.StatusSeeOther)
}
