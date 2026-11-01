package api

import (
	"errors"
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/service/payment"
)

// BillingHandler serves the subscription of a workspace, checkout, and the
// provider billing management. Only members who may manage billing reach
// them, like the web billing page.
type BillingHandler struct {
	paymentService   payment.Provider
	workspaceService *service.WorkspaceService
}

func NewBillingHandler(paymentService payment.Provider, workspaceService *service.WorkspaceService) *BillingHandler {
	return &BillingHandler{paymentService: paymentService, workspaceService: workspaceService}
}

// ErrBillingPermission is returned to members whose role may not manage
// billing.
var ErrBillingPermission = errors.New("workspace billing permission required")

// Billing returns the workspace subscription, its access, and the
// purchasable plans.
func (h *BillingHandler) Billing(w http.ResponseWriter, r *http.Request) {
	billing, err := NewWorkspaceBilling(ctxkeys.WorkspaceMember(r.Context()), ctxkeys.Subscription(r.Context()))
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, billing)
}

// NewWorkspaceBilling returns the billing summary of a workspace for a member
// who may manage billing, and ErrBillingPermission for everyone else.
func NewWorkspaceBilling(member *model.WorkspaceMember, subscription *model.Subscription) (Billing, error) {
	if !member.Can(model.WorkspacePermissionManageBilling) {
		return Billing{}, ErrBillingPermission
	}
	return NewBilling(subscription), nil
}

// CreateCheckout starts a provider checkout for the workspace and returns
// its URL.
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	if !canManageBilling(w, r) {
		return
	}
	var request checkoutRequest
	if !Decode(w, r, &request) {
		return
	}
	if !checkoutSelection(w, &request) {
		return
	}

	checkoutURL, err := h.paymentService.CreateCheckoutURL(workspace.ID, workspace.Slug, request.PlanID, request.Interval, user.Email, workspace.Name)
	if err != nil {
		if errors.Is(err, repository.ErrCheckoutPending) || errors.Is(err, repository.ErrCheckoutSelection) || errors.Is(err, repository.ErrSubscriptionAlreadyPaid) {
			ServiceError(w, err)
			return
		}
		slog.Error("failed to create checkout", "error", err, "workspace_id", workspace.ID, "plan_id", request.PlanID, "provider", h.paymentService.Name())
		Error(w, http.StatusInternalServerError, "internal", "Checkout could not be started. Please try again.")
		return
	}

	h.recordBillingAction(workspace.ID, user.ID, "billing.checkout_started")
	slog.Info("checkout created", "workspace_id", workspace.ID, "provider", h.paymentService.Name())
	JSON(w, http.StatusOK, URLResponse{URL: checkoutURL})
}

// BillingManagement returns the URL of the provider billing management for
// the workspace. A workspace without a provider customer gets 409
// no_billing_account.
func (h *BillingHandler) BillingManagement(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	workspace := ctxkeys.Workspace(r.Context())
	if !canManageBilling(w, r) {
		return
	}
	if !hasBillingAccount(ctxkeys.Subscription(r.Context())) {
		noBillingAccount(w)
		return
	}

	managementURL, err := h.paymentService.BillingManagementURL(workspace.ID, workspace.Slug)
	if err != nil {
		slog.Error("failed to get billing management URL", "error", err, "workspace_id", workspace.ID, "provider", h.paymentService.Name())
		Error(w, http.StatusInternalServerError, "internal", "Billing management could not be opened. Please try again.")
		return
	}
	h.recordBillingAction(workspace.ID, user.ID, "billing.management_opened")
	JSON(w, http.StatusOK, URLResponse{URL: managementURL})
}

func (h *BillingHandler) recordBillingAction(workspaceID, userID, action string) {
	err := h.workspaceService.RecordBillingAction(workspaceID, userID, action)
	if err != nil {
		slog.Warn("failed to record billing audit event", "error", err, "workspace_id", workspaceID)
	}
}

// canManageBilling answers 403 forbidden for members whose role may not
// manage billing.
func canManageBilling(w http.ResponseWriter, r *http.Request) bool {
	if ctxkeys.WorkspaceMember(r.Context()).Can(model.WorkspacePermissionManageBilling) {
		return true
	}
	ServiceError(w, ErrBillingPermission)
	return false
}
