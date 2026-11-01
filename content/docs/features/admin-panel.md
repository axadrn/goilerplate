---
title: "Admin Panel"
description: "Global operator metrics. Support actions. Read-only impersonation."
order: 4
---

## TLDR

The Admin panel is for the people operating the SaaS.

The Workspace Admin role is for a customer's workspace. They are different systems.

## Access

Operators come only from a server-side allowlist of verified email addresses:

```env
ADMIN_EMAILS=admin@example.com,support@example.com
```

An empty value disables the panel. A signed-in non-operator receives not found at `/admin`.

There is no database flag that a user can change to become an operator.

### First operator

Operators are ordinary user accounts. The panel reuses the app's login, sessions, and password recovery.

1. Set `ADMIN_EMAILS` in `.env` to your own address. The generated `.env.example` ships a placeholder that nobody can sign in with.
2. Sign up in the app with that address and open the email verification link.
3. Sign in and visit `/admin`.

## Included operations

- Search and paginate users.
- View user and workspace counts. See paid workspaces and active sessions too.
- Revoke another account's sessions.
- Inspect operator audit history.
- Start a bounded support impersonation session.

## Read-only impersonation

Impersonation creates a separate database session and cookie with a hard one-hour limit. The original operator session remains available for return.

The impersonated session is never an operator. It may view product pages. It cannot change product data or account state. It cannot touch credentials, billing, OAuth, or tokens.

A persistent banner identifies the session and provides the stop action.

## Operator audit

Operator events live in a dedicated audit table without a tenant foreign key. Workspace deletion therefore cannot erase security and billing evidence.

The default retention is 180 days. Event metadata contains identifiers and state. It never contains passwords, tokens, checkout URLs, or provider secrets.

Customer-visible workspace activity stays separate and is deleted with the customer workspace.
