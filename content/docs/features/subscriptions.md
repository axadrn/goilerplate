---
title: "Subscriptions and Billing"
description: "Provider adapters feed one local access model"
order: 2
---

## TLDR

Pick one provider during generation:

- Stripe.
- Polar.
- Creem.

Only that provider enters the project. All adapters produce the same local subscription states. Product access remains provider-agnostic.

## What is included

- Hosted checkout.
- Signed webhook verification.
- Local subscription state.
- Monthly and yearly intervals.
- Provider-hosted billing management.
- Cancellation and grace-period synchronization.
- Idempotent, order-safe provider synchronization.
- Central plan and feature catalog.
- Freemium, trial, and paid-only signup policies.
- Feature, resource, seat, quota, and lifecycle gates.

## Stripe

Generated Stripe Checkout Sessions enable [Stripe Managed Payments](https://docs.stripe.com/payments/managed-payments) in code.

You must also activate Managed Payments and accept its terms in your own Stripe Dashboard. Follow Stripe's current [setup documentation](https://docs.stripe.com/payments/managed-payments/set-up). goilerplate does not duplicate dashboard steps that Stripe may change.

The in-app Manage subscription button creates a short-lived Stripe Customer Portal session. Customers change plans, payment methods, invoices, and cancellations there. The generated app does not duplicate those controls.

Required configuration:

```env
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_ID_PRO_MONTHLY=price_xxx
STRIPE_PRICE_ID_PRO_YEARLY=price_xxx
STRIPE_PRICE_ID_ENTERPRISE_MONTHLY=price_xxx
STRIPE_PRICE_ID_ENTERPRISE_YEARLY=price_xxx
```

## Polar

Polar is useful when you want a developer-focused Merchant of Record. Test and production resources are separate. Before going live, create the production products and prices. Add the production API keys and webhook secrets too.

The in-app Manage subscription button creates a short-lived Polar Customer Portal session for the selected workspace. Customers change plans, payment methods, invoices, and cancellations there. The generated app does not duplicate those controls.

## Creem

Select Creem with `goilerplate new --payment creem` or in the builder. In [Creem Test Mode](https://docs.creem.io/getting-started/test-mode), create one recurring product per plan and interval you offer. The default catalog has Pro and Enterprise, each with monthly and yearly billing. Set up the same products in live mode before going to production.

Register `https://your-app.example/webhooks/payment` in Creem and copy the signing secret. If you select individual [webhook events](https://docs.creem.io/code/webhooks), enable `checkout.completed`, `subscription.active`, `subscription.paid`, `subscription.trialing`, `subscription.update`, `subscription.scheduled_cancel`, `subscription.canceled`, `subscription.past_due`, `subscription.unpaid`, `subscription.expired`, and `subscription.paused`.

```env
CREEM_API_KEY=creem_test_xxx
CREEM_WEBHOOK_SECRET=your_signing_secret
CREEM_PRODUCT_ID_PRO_MONTHLY=prod_pro_monthly_xxx
CREEM_PRODUCT_ID_PRO_YEARLY=prod_pro_yearly_xxx
CREEM_PRODUCT_ID_ENTERPRISE_MONTHLY=prod_enterprise_monthly_xxx
CREEM_PRODUCT_ID_ENTERPRISE_YEARLY=prod_enterprise_yearly_xxx
```

Development uses the test API, and production uses the live API. Set `CREEM_SANDBOX_MODE` to override the default. Use credentials, webhook secrets, and product IDs from the same mode.

The in-app Manage subscription button opens the [Creem Customer Portal](https://docs.creem.io/features/customer-portal). Customers sign in with the email they used at checkout to manage cards, view invoices, and cancel. Enable self-service upgrades in a Creem product bundle if you want customers to switch plans there.

## Local source of truth

The application database decides current access. Product requests never call the provider to ask whether a user is paid.

Webhooks update local state. Every update carries provider identity and ordering data. Older events lose. Duplicate events do nothing. Conflicting events with the same provider timestamp fail closed for reconciliation.

## Plans are product code

The generated app includes a working catalog. Free. Pro. Enterprise. Prices, benefits, feature gates, resource limits, quota limits, and seat limits live together:

```text
internal/access/catalog.go
```

Replace that catalog with your product's real plans. The marketing pricing view and in-app billing view read the same source.

See [Access Control](/docs/features/access-control) for signup policies, service checks, atomic limits, quotas, and lifecycle behavior.

## Safe checkout retries

Only one matching checkout attempt may be live for a workspace. Repeated clicks reuse the same hosted URL. A different plan returns a conflict. So does a different interval or provider. The app never silently charges the wrong choice.

Ambiguous network failures keep a conservative deletion lock long enough for provider webhook retries. Definitive provider rejection releases only the exact attempt.
