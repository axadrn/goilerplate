package access

import (
	"errors"
	"testing"
	"time"
)

func TestResolveSubscriptionAccess(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)
	recentStateChange := now.Add(-24 * time.Hour)
	oldStateChange := now.AddDate(0, 0, -8)
	tests := []struct {
		name string
		sub  Subscription
		want State
	}{
		{name: "free", sub: Subscription{PlanID: PlanFree, Status: StatusActive}, want: StateFree},
		{name: "trial", sub: Subscription{PlanID: PlanPro, Status: StatusTrialing, CurrentPeriodEnd: &future}, want: StateTrial},
		{name: "trial without end falls back", sub: Subscription{PlanID: PlanPro, Status: StatusTrialing}, want: StateFree},
		{name: "paid", sub: Subscription{PlanID: PlanPro, Status: StatusActive}, want: StatePaid},
		{name: "past due grace", sub: Subscription{PlanID: PlanPro, Status: StatusPastDue, StatusChangedAt: &recentStateChange}, want: StateGrace},
		{name: "past due after grace", sub: Subscription{PlanID: PlanPro, Status: StatusPastDue, StatusChangedAt: &oldStateChange}, want: StateFree},
		{name: "past due without state time falls back", sub: Subscription{PlanID: PlanPro, Status: StatusPastDue}, want: StateFree},
		{name: "cancelled grace", sub: Subscription{PlanID: PlanPro, Status: StatusCancelled, CurrentPeriodEnd: &future}, want: StateGrace},
		{name: "expired falls back to free", sub: Subscription{PlanID: PlanPro, Status: StatusCancelled, CurrentPeriodEnd: &past}, want: StateFree},
		{name: "inactive falls back to free", sub: Subscription{PlanID: PlanPro, Status: StatusInactive}, want: StateFree},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Resolve(test.sub, now).State; got != test.want {
				t.Fatalf("state = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPaidOnlyFallbackBlocksAccess(t *testing.T) {
	previous := Product.Lifecycle
	Product.Lifecycle.PaidFallback = FallbackLocked
	t.Cleanup(func() { Product.Lifecycle = previous })

	current := Resolve(Subscription{PlanID: PlanPro, Status: StatusInactive}, time.Now())
	if current.State != StateBlocked {
		t.Fatalf("state = %q, want %q", current.State, StateBlocked)
	}
}

func TestInitialSubscriptionFollowsSignupPolicy(t *testing.T) {
	previous := Product.Signup
	t.Cleanup(func() { Product.Signup = previous })
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	Product.Signup.Mode = SignupFreemium
	if initial := Initial(now); initial.PlanID != PlanFree || initial.Status != StatusActive {
		t.Fatalf("freemium initial = %#v", initial)
	}

	Product.Signup.Mode = SignupTrial
	Product.Signup.TrialPlanID = PlanPro
	Product.Signup.TrialDays = 7
	if initial := Initial(now); initial.PlanID != PlanPro || initial.Status != StatusTrialing || initial.CurrentPeriodEnd == nil || !initial.CurrentPeriodEnd.Equal(now.AddDate(0, 0, 7)) {
		t.Fatalf("trial initial = %#v", initial)
	}

	Product.Signup.Mode = SignupPaidOnly
	if initial := Initial(now); initial.PlanID != PlanFree || initial.Status != StatusBlocked {
		t.Fatalf("paid-only initial = %#v", initial)
	}
}

func TestInitialSubscriptionBlocksUnknownTrialPlan(t *testing.T) {
	previous := Product.Signup
	t.Cleanup(func() { Product.Signup = previous })
	Product.Signup.Mode = SignupTrial
	Product.Signup.TrialPlanID = "missing"

	initial := Initial(time.Now())
	if initial.Status != StatusBlocked {
		t.Fatalf("initial = %#v, want blocked", initial)
	}
}

func TestInitialSubscriptionBlocksUnknownSignupMode(t *testing.T) {
	previous := Product.Signup
	t.Cleanup(func() { Product.Signup = previous })
	Product.Signup.Mode = "typo"

	initial := Initial(time.Now())
	if initial.Status != StatusBlocked {
		t.Fatalf("initial = %#v, want blocked", initial)
	}
}

func TestFeaturesLimitsAndQuotas(t *testing.T) {
	const reports Feature = "reports"
	const uploads Limit = "uploads"
	const exports Quota = "exports"
	current := Access{
		State: StatePaid,
		plan: Plan{
			Features: []Feature{reports},
			Limits:   map[Limit]int{uploads: 3},
			Quotas:   map[Quota]int{exports: 10},
		},
	}
	if err := current.Require(reports); err != nil {
		t.Fatalf("require reports: %v", err)
	}
	if err := current.Require("missing"); !errors.Is(err, ErrFeatureUnavailable) {
		t.Fatalf("missing feature error = %v", err)
	}
	if err := current.RequireWithin(uploads, 2); err != nil {
		t.Fatalf("require capacity: %v", err)
	}
	if !current.HasCapacity(uploads, 2) {
		t.Fatal("capacity is hidden from the UI read path")
	}
	if err := current.RequireWithin(uploads, 3); !errors.Is(err, ErrLimitReached) {
		t.Fatalf("limit error = %v", err)
	}
	if current.HasCapacity(uploads, 3) {
		t.Fatal("capacity exceeds its limit")
	}
	allowance := current.Quota(exports, 9, time.Now().Add(time.Hour))
	if !allowance.Allows(1) || allowance.Remaining() != 1 {
		t.Fatalf("allowance = %#v", allowance)
	}
	if err := allowance.Require(1); err != nil {
		t.Fatalf("require final export: %v", err)
	}
	if err := allowance.Require(2); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("quota error = %v", err)
	}
	if allowance.Allows(2) {
		t.Fatal("allowance permits usage beyond its limit")
	}
}

func TestProductCatalogIsComplete(t *testing.T) {
	switch Product.Signup.Mode {
	case SignupFreemium, SignupTrial, SignupPaidOnly:
	default:
		t.Fatalf("invalid signup mode %q", Product.Signup.Mode)
	}
	for _, check := range []struct {
		name     string
		fallback Fallback
	}{
		{name: "trial", fallback: Product.Signup.TrialFallback},
		{name: "paid", fallback: Product.Lifecycle.PaidFallback},
	} {
		if check.fallback != FallbackFree && check.fallback != FallbackLocked {
			t.Fatalf("invalid %s fallback %q", check.name, check.fallback)
		}
	}
	if _, exists := PlanByID(PlanFree); !exists {
		t.Fatal("catalog has no Free plan")
	}
	if _, exists := PlanByID(Product.Signup.TrialPlanID); !exists || Product.Signup.TrialDays <= 0 {
		t.Fatalf("invalid trial policy = %#v", Product.Signup)
	}
	if Product.Lifecycle.PastDueGraceDays < 0 {
		t.Fatalf("invalid lifecycle policy = %#v", Product.Lifecycle)
	}
	seen := make(map[string]bool, len(Product.Plans))
	for _, plan := range Product.Plans {
		if plan.ID == "" || plan.Name == "" || plan.Description == "" || len(plan.Benefits) == 0 {
			t.Fatalf("incomplete plan = %#v", plan)
		}
		if seen[plan.ID] {
			t.Fatalf("duplicate plan ID %q", plan.ID)
		}
		seen[plan.ID] = true
		if plan.ID != PlanFree && (!plan.Purchasable || plan.MonthlyPrice <= 0 || plan.YearlyPrice <= 0) {
			t.Fatalf("paid plan is not purchasable = %#v", plan)
		}
		if limit, configured := plan.Limits[LimitMembers]; configured && limit < 1 {
			t.Fatalf("invalid member limit = %#v", plan)
		}
	}
}
