package access

import (
	"errors"
	"fmt"
	"time"
)

type Feature string
type Limit string
type Quota string
type State string
type SignupMode string
type Fallback string

const Unlimited = -1

const (
	StateFree    State = "free"
	StateTrial   State = "trial"
	StatePaid    State = "paid"
	StateGrace   State = "grace"
	StateBlocked State = "blocked"
)

const (
	SignupFreemium SignupMode = "freemium"
	SignupTrial    SignupMode = "trial"
	SignupPaidOnly SignupMode = "paid_only"
)

const (
	FallbackFree   Fallback = "free"
	FallbackLocked Fallback = "locked"
)

const (
	StatusActive     = "active"
	StatusTrialing   = "trialing"
	StatusPastDue    = "past_due"
	StatusIncomplete = "incomplete"
	StatusInactive   = "inactive"
	StatusCancelled  = "cancelled"
	StatusBlocked    = "blocked"
)

const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

const FeaturePrioritySupport Feature = "priority_support"

const LimitMembers Limit = "members"

var (
	ErrProductAccess      = errors.New("product access is unavailable")
	ErrFeatureUnavailable = errors.New("feature is unavailable")
	ErrLimitReached       = errors.New("limit reached")
	ErrQuotaExceeded      = errors.New("quota exceeded")
)

type SignupPolicy struct {
	Mode          SignupMode
	TrialPlanID   string
	TrialDays     int
	TrialFallback Fallback
}

type LifecyclePolicy struct {
	PastDueGraceDays int
	PaidFallback     Fallback
}

type Plan struct {
	ID           string
	Name         string
	Description  string
	MonthlyPrice int
	YearlyPrice  int
	Features     []Feature
	Limits       map[Limit]int
	Quotas       map[Quota]int
	Benefits     []string
	Purchasable  bool
}

type Subscription struct {
	PlanID           string
	Status           string
	CurrentPeriodEnd *time.Time
	StatusChangedAt  *time.Time
}

type InitialSubscription struct {
	PlanID           string
	Status           string
	CurrentPeriodEnd *time.Time
}

type Access struct {
	PlanID       string
	State        State
	TrialEndsAt  *time.Time
	AccessEndsAt *time.Time
	plan         Plan
}

type Allowance struct {
	Limit    int
	Used     int
	ResetsAt time.Time
}

func Resolve(subscription Subscription, now time.Time) Access {
	plan, exists := PlanByID(subscription.PlanID)
	if !exists {
		return blockedAccess()
	}

	switch subscription.Status {
	case StatusTrialing:
		if subscription.CurrentPeriodEnd != nil && now.Before(*subscription.CurrentPeriodEnd) {
			return Access{PlanID: plan.ID, State: StateTrial, TrialEndsAt: subscription.CurrentPeriodEnd, plan: plan}
		}
		return fallbackAccess(Product.Signup.TrialFallback)
	case StatusActive:
		if plan.ID == PlanFree {
			return Access{PlanID: plan.ID, State: StateFree, plan: plan}
		}
		return Access{PlanID: plan.ID, State: StatePaid, plan: plan}
	case StatusPastDue:
		graceEndsAt := pastDueGraceEndsAt(subscription.StatusChangedAt)
		if plan.ID != PlanFree && graceEndsAt != nil && now.Before(*graceEndsAt) {
			return Access{PlanID: plan.ID, State: StateGrace, AccessEndsAt: graceEndsAt, plan: plan}
		}
		return fallbackAccess(Product.Lifecycle.PaidFallback)
	case StatusCancelled:
		if plan.ID != PlanFree && subscription.CurrentPeriodEnd != nil && now.Before(*subscription.CurrentPeriodEnd) {
			return Access{PlanID: plan.ID, State: StateGrace, AccessEndsAt: subscription.CurrentPeriodEnd, plan: plan}
		}
		return fallbackAccess(Product.Lifecycle.PaidFallback)
	case StatusIncomplete, StatusInactive:
		return fallbackAccess(Product.Lifecycle.PaidFallback)
	case StatusBlocked:
		return blockedAccess()
	default:
		return blockedAccess()
	}
}

func Initial(now time.Time) InitialSubscription {
	switch Product.Signup.Mode {
	case SignupTrial:
		if _, exists := PlanByID(Product.Signup.TrialPlanID); !exists || Product.Signup.TrialDays <= 0 {
			return InitialSubscription{PlanID: PlanFree, Status: StatusBlocked}
		}
		endsAt := now.AddDate(0, 0, Product.Signup.TrialDays)
		return InitialSubscription{PlanID: Product.Signup.TrialPlanID, Status: StatusTrialing, CurrentPeriodEnd: &endsAt}
	case SignupPaidOnly:
		return InitialSubscription{PlanID: PlanFree, Status: StatusBlocked}
	case SignupFreemium:
		return InitialSubscription{PlanID: PlanFree, Status: StatusActive}
	default:
		return InitialSubscription{PlanID: PlanFree, Status: StatusBlocked}
	}
}

func (a Access) RequireProduct() error {
	if a.State == StateBlocked {
		return ErrProductAccess
	}
	return nil
}

func (a Access) Has(feature Feature) bool {
	if a.State == StateBlocked {
		return false
	}
	for _, candidate := range a.plan.Features {
		if candidate == feature {
			return true
		}
	}
	return false
}

func (a Access) Require(feature Feature) error {
	if !a.Has(feature) {
		return fmt.Errorf("%w: %s", ErrFeatureUnavailable, feature)
	}
	return nil
}

func (a Access) Limit(limit Limit) int {
	if a.State == StateBlocked {
		return 0
	}
	return a.plan.Limits[limit]
}

func (a Access) RequireWithin(limit Limit, used int) error {
	if !a.HasCapacity(limit, used) {
		return fmt.Errorf("%w: %s", ErrLimitReached, limit)
	}
	return nil
}

func (a Access) HasCapacity(limit Limit, used int) bool {
	maximum := a.Limit(limit)
	return used >= 0 && (maximum == Unlimited || used < maximum)
}

func (a Access) Quota(quota Quota, used int, resetsAt time.Time) Allowance {
	if a.State == StateBlocked {
		return Allowance{Used: used, ResetsAt: resetsAt}
	}
	return Allowance{Limit: a.plan.Quotas[quota], Used: used, ResetsAt: resetsAt}
}

func (a Allowance) Require(amount int) error {
	if !a.Allows(amount) {
		return ErrQuotaExceeded
	}
	return nil
}

func (a Allowance) Allows(amount int) bool {
	if amount < 0 || a.Used < 0 {
		return false
	}
	return a.Limit == Unlimited || (a.Used <= a.Limit && amount <= a.Limit-a.Used)
}

func (a Allowance) Remaining() int {
	if a.Used < 0 {
		return 0
	}
	if a.Limit == Unlimited {
		return Unlimited
	}
	if a.Used >= a.Limit {
		return 0
	}
	return a.Limit - a.Used
}

func fallbackAccess(fallback Fallback) Access {
	if fallback == FallbackFree {
		plan, _ := PlanByID(PlanFree)
		return Access{PlanID: plan.ID, State: StateFree, plan: plan}
	}
	return blockedAccess()
}

func blockedAccess() Access {
	return Access{PlanID: PlanFree, State: StateBlocked}
}

func pastDueGraceEndsAt(statusChangedAt *time.Time) *time.Time {
	if statusChangedAt == nil || Product.Lifecycle.PastDueGraceDays <= 0 {
		return nil
	}
	endsAt := statusChangedAt.AddDate(0, 0, Product.Lifecycle.PastDueGraceDays)
	return &endsAt
}
