---
title: "Security"
description: "Concrete browser protection. Sessions. Input. Tenants. Providers."
order: 4
---

## TLDR

goilerplate ships conservative defaults for the common ways a SaaS gets hurt.

Security is still your responsibility after you change the product.

## Browser boundary

Every response sets:

- Content Security Policy. htmx and Datastar pages use a per-request nonce, the SvelteKit app uses a hash, and JSON answers allow nothing.
- `X-Content-Type-Options: nosniff`.
- `Referrer-Policy`.
- `Permissions-Policy`.
- Frame restrictions.

In production, cookies are `Secure`. Set HSTS at your HTTPS proxy.

Generated frontend assets are local. Optional analytics hosts enter the policy only through typed configuration.

## CSRF

Every state-changing browser form includes a random CSRF token that must match its cookie. Unsafe requests without a valid token are rejected before the handler runs.

Provider webhooks use their own HMAC signatures and do not use browser CSRF tokens.

## Sessions

- Random opaque browser token.
- Hash stored in the database.
- Secure, HTTP-only, same-site cookie.
- Server-side expiry and revocation.
- Session replacement after recovery and sensitive credential changes.
- Indexed cleanup of expired rows.

See [Authentication](/docs/features/authentication).

## Input limits and validation

Request bodies use explicit maximum sizes before parsing. Central validators check names and emails. They also check identifiers, slugs, paths, and provider values.

Validation happens before trimming when control characters matter.

Templates escape untrusted output by default. Signed URLs and secrets are never placed in logs.

## Rate limits

Abuse-sensitive paths have separate limits. This includes authentication and email delivery. Account email changes and workspace invitations. Each limit uses the right identity for that operation.

Reverse proxy headers are trusted only when the direct peer belongs to configured trusted CIDRs. Untrusted clients cannot choose their own rate-limit key through `X-Forwarded-For`.

The included limiter is process-local. Use a shared limiter before running multiple application replicas.

## Tenant isolation

With Workspaces selected, middleware resolves the workspace and membership before product handlers run. Repositories require `workspace_id` for tenant data.

Cross-tenant access returns not found. Role changes and final-Owner checks are revalidated inside the same locked transaction as the mutation.

## Billing webhooks

- Constant-time signature verification.
- Bounded request bodies.
- Replay tolerance where the provider supplies signed timestamps.
- Exact provider object binding.
- Duplicate event protection.
- Atomic ordering checks.
- Fail-closed behavior for conflicting states.

## File uploads

Storage records ownership metadata and validates upload size and content type. The service generates object keys instead of accepting them from browser input. Calling handlers must verify authorization before issuing short-lived signed access to private files.

## Admin boundary

Global operator access comes only from a server-side verified-email allowlist. Read-only impersonation uses a separate short-lived session and cannot chain into operator access.

See [Admin Panel](/docs/features/admin-panel).

## What to review before launch

1. Run `go test -race ./...`.
2. Review every new state-changing route for authentication, authorization, CSRF, limits, and audit needs.
3. Verify production secrets and callback origins.
4. Test account recovery, role races, billing retries, and cross-tenant access.
5. Keep dependencies and the Go toolchain current.

These defaults reduce common mistakes. They are not a compliance certificate or penetration test.
