package model

import (
	"testing"
	"time"
)

func TestSubscriptionIsActiveDuringCancelledPaidPeriod(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	recentStateChange := now.Add(-time.Hour)

	tests := []struct {
		name string
		sub  Subscription
		want bool
	}{
		{name: "free", sub: Subscription{PlanID: SubscriptionPlanFree, Status: SubscriptionStatusActive}, want: true},
		{name: "active", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusActive}, want: true},
		{name: "trialing", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusTrialing, CurrentPeriodEnd: &future}, want: true},
		{name: "trialing without end", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusTrialing}, want: false},
		{name: "past due", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusPastDue, ProviderStateUpdatedAt: &recentStateChange}, want: true},
		{name: "past due without state time", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusPastDue}, want: false},
		{name: "incomplete", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusIncomplete}, want: false},
		{name: "inactive with future period", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusInactive, CurrentPeriodEnd: &future}, want: false},
		{name: "cancelled with remaining period", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusCancelled, CurrentPeriodEnd: &future}, want: true},
		{name: "cancelled after period", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusCancelled, CurrentPeriodEnd: &past}, want: false},
		{name: "cancelled without period", sub: Subscription{PlanID: SubscriptionPlanPro, Status: SubscriptionStatusCancelled}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.sub.IsActiveAt(now); got != test.want {
				t.Fatalf("IsActiveAt() = %v, want %v", got, test.want)
			}
		})
	}
}
