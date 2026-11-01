---
title: "Access Control"
description: "Freemium, trials, paid access, feature gates, limits, seats, quotas, and lifecycle rules"
order: 3
---

## TLDR

Payments confirm what happened at the provider. The generated access package decides what that unlocks in your product.

One local catalog defines plans, features, resource limits, seat limits, quota limits, and signup policy. Stripe and Polar both update the same local subscription model. Product services never depend on either provider.

## The two layers

The product model answers how accounts begin:

- Freemium starts on Free.
- Trial starts on Pro for a configured number of days.
- Paid-only starts locked until checkout succeeds.

Access rules then answer what the current account may do:

- Feature gates allow a capability such as CSV export.
- Resource limits cap stored objects such as uploads.
- Seat limits cap workspace members.
- Periodic quotas cap usage during a time window.
- Lifecycle rules handle trials, active subscriptions, grace periods, cancellations, and blocked access.

These layers combine without a policy language or runtime framework.

## One server-side value

htmx 4.0 and Datastar 1.0 render on the server. Templates and services therefore use the same `access.Access` value.

The template mirrors access for a good interface:

```go
if currentAccess.Has(access.FeatureProjectExport) {
    // Render Export CSV.
}
```

The service enforces access with an ordinary early return:

```go
func (s *ProjectService) Export(workspaceID string) ([]byte, error) {
    currentAccess, err := s.subscriptions.Access(workspaceID)
    if err != nil {
        return nil, err
    }
    if err := currentAccess.RequireProduct(); err != nil {
        return nil, err
    }
    if err := currentAccess.Require(access.FeatureProjectExport); err != nil {
        return nil, err
    }
    return s.exportProjects(workspaceID)
}
```

Multiple requirements stay explicit:

```go
if err := currentAccess.RequireProduct(); err != nil {
    return err
}
if err := currentAccess.Require(access.FeatureProjectExport); err != nil {
    return err
}
```

The browser never grants access. Hiding a button is only presentation.

## Product catalog

Edit `internal/access/catalog.go` to change plans or policy. This is normal Go code:

```go
var Product = Catalog{
    Signup: SignupPolicy{
        Mode:          SignupFreemium,
        TrialPlanID:   PlanPro,
        TrialDays:     7,
        TrialFallback: FallbackFree,
    },
    Lifecycle: LifecyclePolicy{
        PastDueGraceDays: 7,
        PaidFallback:     FallbackFree,
    },
    Plans: []Plan{
        // Free, Pro, Enterprise.
    },
}
```

Changing `Mode` changes new signups. It does not silently rewrite existing subscriptions.

Common policies:

```go
// Freemium. Paid subscriptions fall back to Free.
Signup: SignupPolicy{Mode: SignupFreemium},
Lifecycle: LifecyclePolicy{
    PastDueGraceDays: 7,
    PaidFallback:     FallbackFree,
},

// Seven-day trial, then lock until payment.
Signup: SignupPolicy{
    Mode:          SignupTrial,
    TrialPlanID:   PlanPro,
    TrialDays:     7,
    TrialFallback: FallbackLocked,
},
Lifecycle: LifecyclePolicy{
    PastDueGraceDays: 7,
    PaidFallback:     FallbackLocked,
},

// Payment required before product access.
Signup: SignupPolicy{Mode: SignupPaidOnly},
Lifecycle: LifecyclePolicy{
    PastDueGraceDays: 7,
    PaidFallback:     FallbackLocked,
},
```

Signup policy controls only how new accounts begin. Lifecycle policy controls existing subscriptions, including the past-due grace period and what happens when paid access ends.

## Feature gates

Declare a typed feature, add it to the plans that include it, then require it in the service:

```go
const FeatureProjectExport access.Feature = "project_export"

if err := currentAccess.Require(FeatureProjectExport); err != nil {
    return err
}
```

These are product-entitlement gates. They answer whether this account may use a paid capability. Percentage rollouts, experiments, and operational kill switches are a separate concern and should not decide billing access.

## Resource and seat limits

The UI may show `3 of 5 seats`, but enforcement belongs in the database transaction that accepts an invitation. A separate count followed by an insert can be bypassed by concurrent requests.

Templates use `currentAccess.HasCapacity(LimitMembers, currentSeats)`. Services and repositories use `RequireWithin` and the transactional check. Both call the same small rule, but the database transaction remains the final authority.

A repository that creates a limited resource serializes creation for one owner, counts current rows, checks the configured limit, inserts, and commits in one transaction. The access rule stays the same with PostgreSQL. Use the database transaction primitive native to the selected database.

Seats use the generic `Limit` primitive. Invitation acceptance applies the transactional pattern, so there is no second seat-specific access system to learn.

## Lifecycle

Trials require a real end time. Missing lifecycle data fails closed instead of granting indefinite access.

Past-due access lasts for `Lifecycle.PastDueGraceDays`, measured from the locally stored provider-state timestamp. Cancellation keeps access only until the stored billing period ends. After either window closes, `Lifecycle.PaidFallback` decides whether the account becomes Free or Blocked.

## Periodic quotas

A quota combines a configured maximum with stored usage and a reset time:

```go
allowance := currentAccess.Quota(QuotaExports, usage.Count, usage.ResetsAt)
if allowance.Allows(1) {
    // Render the available action and allowance.Remaining().
}
if err := allowance.Require(1); err != nil {
    return err
}
```

Persist the usage increment in the same transaction as the operation it protects. Add a quota only when the product has a real metered action.

## Provider adapters

Provider adapters translate external names into local states:

```text
Stripe status -> local subscription state
Polar status  -> local subscription state
                     |
                     v
                access.Access
```

Adding another payment provider requires a new adapter. Feature gates, limits, services, and templates do not change.

## Public APIs later

The service architecture is ready for JSON handlers or mobile clients. A future API response can describe current access, but every API operation must still call the same services. Do not let a client-provided plan, feature, or limit decide authorization.
