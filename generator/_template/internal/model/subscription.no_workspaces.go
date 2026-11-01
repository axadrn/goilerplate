package model

import (
	"fmt"
	"time"

	"{{.ModulePath}}/internal/access"
)

type Subscription struct {
	ID                            string     `db:"id"`
	UserID                        string     `db:"user_id"`
	PlanID                        string     `db:"plan_id"`
	Status                        string     `db:"status"`
	Provider                      string     `db:"provider"`
	ProviderCustomerID            *string    `db:"provider_customer_id"`
	ProviderSubscriptionID        *string    `db:"provider_subscription_id"`
	ProviderSubscriptionCreatedAt *time.Time `db:"provider_subscription_created_at"`
	ProviderStateUpdatedAt        *time.Time `db:"provider_state_updated_at"`
	ProviderPaymentUpdatedAt      *time.Time `db:"provider_payment_updated_at"`
	CurrentPeriodEnd              *time.Time `db:"current_period_end"`
	Amount                        *int       `db:"amount"`
	Currency                      string     `db:"currency"`
	Interval                      *string    `db:"interval"`
	CreatedAt                     time.Time  `db:"created_at"`
	UpdatedAt                     time.Time  `db:"updated_at"`
}

const (
	SubscriptionStatusActive     = access.StatusActive
	SubscriptionStatusTrialing   = access.StatusTrialing
	SubscriptionStatusPastDue    = access.StatusPastDue
	SubscriptionStatusIncomplete = access.StatusIncomplete
	SubscriptionStatusInactive   = access.StatusInactive
	SubscriptionStatusCancelled  = access.StatusCancelled
	SubscriptionStatusBlocked    = access.StatusBlocked
)

const (
	ProviderPolar  = "polar"
	ProviderStripe = "stripe"
)

const (
	SubscriptionPlanFree       = access.PlanFree
	SubscriptionPlanPro        = access.PlanPro
	SubscriptionPlanEnterprise = access.PlanEnterprise
)

const (
	SubscriptionIntervalMonthly = "monthly"
	SubscriptionIntervalYearly  = "yearly"
)

func FormatPlanPrice(cents int) string {
	if cents%100 != 0 {
		return fmt.Sprintf("$%.2f", float64(cents)/100)
	}
	return fmt.Sprintf("$%d", cents/100)
}

func (s *Subscription) IsActive() bool {
	return s.IsActiveAt(time.Now())
}

func (s *Subscription) IsActiveAt(now time.Time) bool {
	currentAccess := s.AccessAt(now)
	return currentAccess.State == access.StateTrial ||
		currentAccess.State == access.StatePaid ||
		currentAccess.State == access.StateGrace ||
		(currentAccess.State == access.StateFree && s.PlanID == SubscriptionPlanFree)
}

func (s *Subscription) HasProviderSubscription() bool {
	return s.ProviderSubscriptionID != nil && *s.ProviderSubscriptionID != ""
}

func (s *Subscription) BlocksAccountDeletion(now time.Time) bool {
	return s.HasProviderSubscription() || (s.PlanID != SubscriptionPlanFree && s.IsActiveAt(now))
}

func (s *Subscription) FormatPrice() string {
	if s.Amount == nil || *s.Amount == 0 {
		return ""
	}

	currencySymbols := map[string]string{
		"usd": "$",
		"eur": "€",
		"gbp": "£",
	}

	amount := float64(*s.Amount) / 100.0
	symbol := currencySymbols[s.Currency]
	if symbol == "" {
		symbol = "$"
	}

	interval := "month"
	if s.Interval != nil && *s.Interval == SubscriptionIntervalYearly {
		interval = "year"
	}

	return fmt.Sprintf("%s%.0f/%s", symbol, amount, interval)
}

// HasFeature checks if the subscription has access to a specific feature
func (s *Subscription) HasFeature(feature access.Feature) bool {
	return s.Access().Has(feature)
}

func (s *Subscription) Access() access.Access { return s.AccessAt(time.Now()) }

func (s *Subscription) AccessAt(now time.Time) access.Access {
	statusChangedAt := s.ProviderStateUpdatedAt
	if statusChangedAt == nil && !s.UpdatedAt.IsZero() {
		statusChangedAt = &s.UpdatedAt
	}
	return access.Resolve(access.Subscription{
		PlanID:           s.PlanID,
		Status:           s.Status,
		CurrentPeriodEnd: s.CurrentPeriodEnd,
		StatusChangedAt:  statusChangedAt,
	}, now)
}
