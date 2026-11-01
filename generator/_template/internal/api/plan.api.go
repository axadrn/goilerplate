package api

import (
	"net/http"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
)

// Plan is a purchasable plan of the product catalog with its prices.
type Plan struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Benefits    []string    `json:"benefits"`
	Current     bool        `json:"current"`
	Prices      []PlanPrice `json:"prices"`
}

// PlanPrice is the price of a plan for one billing interval. Amount is in
// the smallest currency unit, DisplayPrice is formatted like the web page.
type PlanPrice struct {
	Interval     string `json:"interval"`
	Amount       int    `json:"amount"`
	DisplayPrice string `json:"display_price"`
}

// NewPlans lists the purchasable plans of access.Product and marks the plan
// of the current access.
func NewPlans(currentPlanID string) []Plan {
	plans := make([]Plan, 0, len(access.Product.Plans))
	for _, plan := range access.Product.Plans {
		if !plan.Purchasable {
			continue
		}
		benefits := plan.Benefits
		if benefits == nil {
			benefits = []string{}
		}
		plans = append(plans, Plan{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Benefits:    benefits,
			Current:     plan.ID == currentPlanID,
			Prices: []PlanPrice{
				{Interval: model.SubscriptionIntervalMonthly, Amount: plan.MonthlyPrice, DisplayPrice: model.FormatPlanPrice(plan.MonthlyPrice)},
				{Interval: model.SubscriptionIntervalYearly, Amount: plan.YearlyPrice, DisplayPrice: model.FormatPlanPrice(plan.YearlyPrice)},
			},
		})
	}
	return plans
}

// Billing is the answer of the billing endpoints: the subscription, its
// resolved access, and the plans a checkout can start.
type Billing struct {
	Subscription *Subscription `json:"subscription"`
	Access       *Access       `json:"access"`
	Plans        []Plan        `json:"plans"`
}

// NewBilling builds the billing answer for a subscription, which may be nil.
func NewBilling(subscription *model.Subscription) Billing {
	billing := Billing{Subscription: NewSubscription(subscription)}
	currentPlanID := ""
	if subscription != nil {
		current := subscription.Access()
		billing.Access = NewAccess(current)
		currentPlanID = current.PlanID
	}
	billing.Plans = NewPlans(currentPlanID)
	return billing
}

// URLResponse carries a provider-hosted URL the client opens next.
type URLResponse struct {
	URL string `json:"url"`
}

type checkoutRequest struct {
	PlanID   string `json:"plan_id"`
	Interval string `json:"interval"`
}

// checkoutSelection validates a checkout request like the web handler: the
// plan must exist and be purchasable, an empty interval means monthly. It
// writes the 422 answer and returns false when the selection is invalid.
func checkoutSelection(w http.ResponseWriter, request *checkoutRequest) bool {
	plan, exists := access.PlanByID(request.PlanID)
	if !exists || !plan.Purchasable {
		Error(w, http.StatusUnprocessableEntity, "invalid_plan", "Choose an available paid plan.")
		return false
	}
	if request.Interval == "" {
		request.Interval = model.SubscriptionIntervalMonthly
	}
	if request.Interval != model.SubscriptionIntervalMonthly && request.Interval != model.SubscriptionIntervalYearly {
		Error(w, http.StatusUnprocessableEntity, "invalid_interval", "Choose monthly or yearly billing.")
		return false
	}
	return true
}

// hasBillingAccount reports whether the provider knows a customer for the
// subscription, which the portal and billing management need.
func hasBillingAccount(subscription *model.Subscription) bool {
	return subscription != nil && subscription.ProviderCustomerID != nil && *subscription.ProviderCustomerID != ""
}

func noBillingAccount(w http.ResponseWriter) {
	Error(w, http.StatusConflict, "no_billing_account", "Start a paid plan before managing billing.")
}
