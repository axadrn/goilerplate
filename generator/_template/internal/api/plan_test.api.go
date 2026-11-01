package api_test

import (
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/model"
)

func TestNewPlansMarksTheCurrentPlan(t *testing.T) {
	plans := api.NewPlans(model.SubscriptionPlanEnterprise)
	expectPurchasablePlansShape(t, plans)
	if plans[0].Current || !plans[1].Current {
		t.Fatalf("current flags = %t, %t", plans[0].Current, plans[1].Current)
	}
}

func expectPurchasablePlans(t *testing.T, plans []api.Plan) {
	t.Helper()
	expectPurchasablePlansShape(t, plans)
	if plans[0].Current || plans[1].Current {
		t.Fatalf("a paid plan is current: %#v", plans)
	}
}

func expectPurchasablePlansShape(t *testing.T, plans []api.Plan) {
	t.Helper()
	if len(plans) != 2 || plans[0].ID != model.SubscriptionPlanPro || plans[1].ID != model.SubscriptionPlanEnterprise {
		t.Fatalf("plans = %#v", plans)
	}
	pro := plans[0]
	if len(pro.Prices) != 2 || pro.Prices[0].Interval != "monthly" || pro.Prices[0].Amount != 2900 || pro.Prices[0].DisplayPrice != "$29" || pro.Prices[1].Interval != "yearly" || pro.Prices[1].DisplayPrice != "$279" {
		t.Fatalf("pro prices = %#v", pro.Prices)
	}
	if len(pro.Benefits) == 0 {
		t.Fatalf("pro = %#v", pro)
	}
}
