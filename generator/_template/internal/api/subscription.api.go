package api

import (
	"time"

	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
)

// Subscription is the public JSON shape of a subscription. Provider
// identifiers and checkout bookkeeping stay on the server.
type Subscription struct {
	ID               string     `json:"id"`
	PlanID           string     `json:"plan_id"`
	Status           string     `json:"status"`
	Provider         string     `json:"provider"`
	CurrentPeriodEnd *time.Time `json:"current_period_end"`
	Amount           *int       `json:"amount"`
	Currency         string     `json:"currency"`
	Interval         *string    `json:"interval"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// NewSubscription converts a subscription model into its public JSON shape.
func NewSubscription(subscription *model.Subscription) *Subscription {
	if subscription == nil {
		return nil
	}
	return &Subscription{
		ID:               subscription.ID,
		PlanID:           subscription.PlanID,
		Status:           subscription.Status,
		Provider:         subscription.Provider,
		CurrentPeriodEnd: utcTime(subscription.CurrentPeriodEnd),
		Amount:           subscription.Amount,
		Currency:         subscription.Currency,
		Interval:         subscription.Interval,
		CreatedAt:        subscription.CreatedAt.UTC(),
		UpdatedAt:        subscription.UpdatedAt.UTC(),
	}
}

// Access is the resolved product access of a subscription: the plan, the
// state, and the trial or grace end.
type Access struct {
	PlanID       string       `json:"plan_id"`
	State        access.State `json:"state"`
	TrialEndsAt  *time.Time   `json:"trial_ends_at"`
	AccessEndsAt *time.Time   `json:"access_ends_at"`
}

// NewAccess converts resolved access into its public JSON shape.
func NewAccess(current access.Access) *Access {
	return &Access{
		PlanID:       current.PlanID,
		State:        current.State,
		TrialEndsAt:  utcTime(current.TrialEndsAt),
		AccessEndsAt: utcTime(current.AccessEndsAt),
	}
}
