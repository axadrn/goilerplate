package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/service/payment"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
)

type BillingHandler struct {
	paymentService   payment.Provider
	workspaceService *service.WorkspaceService
}

func NewBillingHandler(paymentService payment.Provider, workspaceService *service.WorkspaceService) *BillingHandler {
	return &BillingHandler{
		paymentService:   paymentService,
		workspaceService: workspaceService,
	}
}

func (h *BillingHandler) BillingPage(w http.ResponseWriter, r *http.Request) {
	if !ctxkeys.WorkspaceMember(r.Context()).Can(model.WorkspacePermissionManageBilling) {
		w.WriteHeader(http.StatusForbidden)
		render(w, r, pages.ErrorPage(http.StatusForbidden, "Access denied", "Only workspace owners can manage billing."))
		return
	}
	render(w, r, pages.Billing())
}

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	member := ctxkeys.WorkspaceMember(r.Context())
	if !member.Can(model.WorkspacePermissionManageBilling) {
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Action not allowed", Description: "Only workspace owners can start checkout.", Type: toast.TypeError}), "#toast-container")
		} else {
			w.WriteHeader(http.StatusForbidden)
			render(w, r, pages.ErrorPage(http.StatusForbidden, "Action not allowed", "Only workspace owners can start checkout."))
		}
		return
	}

	planID := r.FormValue("plan_id")
	plan, exists := access.PlanByID(planID)
	if !exists || !plan.Purchasable {
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Invalid plan", Description: "Choose an available paid plan.", Type: toast.TypeError}), "#toast-container")
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
	if interval != model.SubscriptionIntervalMonthly && interval != model.SubscriptionIntervalYearly {
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Invalid billing interval", Description: "Choose monthly or yearly billing.", Type: toast.TypeError}), "#toast-container")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			render(w, r, pages.ErrorPage(http.StatusBadRequest, "Invalid billing interval", "Choose monthly or yearly billing."))
		}
		return
	}

	checkoutURL, err := h.paymentService.CreateCheckoutURL(workspace.ID, workspace.Slug, planID, interval, user.Email, workspace.Name)
	if err != nil {
		if errors.Is(err, repository.ErrCheckoutPending) || errors.Is(err, repository.ErrCheckoutSelection) || errors.Is(err, repository.ErrSubscriptionAlreadyPaid) {
			if isAction(r) {
				appendElement(w, r, toast.Toast(toast.Props{Title: "Checkout already active", Description: "Finish the existing checkout or manage the current subscription first.", Type: toast.TypeError}), "#toast-container")
			} else {
				w.WriteHeader(http.StatusConflict)
				render(w, r, pages.ErrorPage(http.StatusConflict, "Checkout already active", "Finish the existing checkout or manage the current subscription first."))
			}
			return
		}
		slog.Error("failed to create checkout", "error", err, "workspace_id", workspace.ID, "plan_id", planID, "provider", h.paymentService.Name())
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Checkout unavailable", Description: "Checkout could not be started. Please try again.", Type: toast.TypeError}), "#toast-container")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Checkout unavailable", "Checkout could not be started. Please try again."))
		}
		return
	}

	if h.workspaceService != nil {
		if err := h.workspaceService.RecordBillingAction(workspace.ID, user.ID, "billing.checkout_started"); err != nil {
			slog.Warn("failed to record billing audit event", "error", err, "workspace_id", workspace.ID)
		}
	}
	slog.Info("redirecting to checkout", "workspace_id", workspace.ID, "provider", h.paymentService.Name())
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

func (h *BillingHandler) BillingManagement(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	if !ctxkeys.WorkspaceMember(r.Context()).Can(model.WorkspacePermissionManageBilling) {
		w.WriteHeader(http.StatusForbidden)
		render(w, r, pages.ErrorPage(http.StatusForbidden, "Action not allowed", "Only workspace owners can manage billing."))
		return
	}

	managementURL, err := h.paymentService.BillingManagementURL(workspace.ID, workspace.Slug)
	if err != nil {
		slog.Error("failed to get billing management URL", "error", err, "workspace_id", workspace.ID, "provider", h.paymentService.Name())
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Billing unavailable", "Billing management could not be opened. Please try again."))
		return
	}
	if h.workspaceService != nil {
		if err := h.workspaceService.RecordBillingAction(workspace.ID, user.ID, "billing.management_opened"); err != nil {
			slog.Warn("failed to record billing audit event", "error", err, "workspace_id", workspace.ID)
		}
	}

	redirect(w, r, managementURL, http.StatusSeeOther)
}
