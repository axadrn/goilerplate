---
title: "JSON API"
description: "JSON endpoints under /api/v1 on the same services as the web app. Bearer or cookie sessions, one error envelope, OpenAPI."
order: 2
---

## TLDR

Add `--api` to an htmx or Datastar project. Pick `--framework svelte` for a SvelteKit app on the API. Or pick `--headless` for the Go backend without a native frontend.

```bash
goilerplate new --module example.com/acme --framework htmx --api ./acme
goilerplate new --module example.com/acme --framework svelte ./acme
goilerplate new --module example.com/acme --headless ./acme
```

Every project with the API gets auth, account, sessions, billing, and admin endpoints. Workspaces and storage add theirs when you select them.

The API lives under `/api/v1`. Every JSON handler calls the same services as the HTML handlers. Rules and permission checks exist once.

## Three ways to get it

| Selection | What you get |
| --- | --- |
| `--framework htmx --api` or `--framework datastar --api` | The web app plus JSON endpoints in the same binary. Good for a mobile app or an integration next to the web app. |
| `--framework svelte` | A [SvelteKit](/docs/guides/sveltekit) app in `web/` on the JSON API. The Go binary embeds it and serves it on the same origin as the API. |
| `--headless` | The Go backend with the JSON API and no native frontend. No templ, Tailwind, or assets. Bring React, Svelte, or a mobile client. |

SvelteKit and Headless always include the API. Neither has blog or docs, because both are HTML products. Everything else stays selectable: database, payments, email, workspaces, OAuth, and storage.

A Headless project answers `GET /` with its name and the API base path, and `GET /healthz` with 204. Every other unknown path returns the JSON error envelope with `not_found`. In a SvelteKit project every other `GET` outside `/api/` is the app, and unknown paths under `/api/` return `not_found`.

## Endpoints

Every API project:

| Area | Routes |
| --- | --- |
| Meta | `GET /api/v1/openapi.yaml`, `GET /api/v1/csrf`, `GET /api/v1/meta` |
| Auth | `POST /api/v1/auth/password`, `magic-link`, `magic-link/verify`, `forgot-password`, `forgot-password/verify`, `email-change/verify`, `onboarding`, `logout` |
| Account | `GET /api/v1/account`, `PATCH /api/v1/account/name`, `PATCH /api/v1/account/email`, `POST /api/v1/account/password`, `POST /api/v1/account/password/set`, `DELETE /api/v1/account/password`, `DELETE /api/v1/account` |
| Sessions | `GET /api/v1/account/sessions`, `DELETE /api/v1/account/sessions/{id}`, `DELETE /api/v1/account/sessions` |
| API tokens | `GET` and `POST /api/v1/account/tokens`, `DELETE /api/v1/account/tokens/{id}` |
| Admin | `GET /api/v1/admin/users`, `POST /api/v1/admin/users/{userID}/sessions/revoke` |

Depending on your selection:

| Module | Routes |
| --- | --- |
| Billing without Workspaces | `GET /api/v1/billing`, `POST /api/v1/billing/checkout`, `GET /api/v1/billing/portal` |
| Storage | `POST /api/v1/account/avatar`, `DELETE /api/v1/account/avatar` |
| Workspaces | `GET` and `POST /api/v1/workspaces`, `GET` and `DELETE /api/v1/workspaces/{workspaceSlug}`, `PATCH .../name`, `POST .../invitations`, `POST .../invitations/{invitationID}/resend`, `DELETE .../invitations/{invitationID}`, `PATCH .../members/{memberID}/role`, `DELETE .../members/{memberID}`, `POST .../leave`, `GET .../activity` |
| Workspace billing | `GET .../billing`, `POST .../billing/checkout`, `GET .../billing/manage` under `/api/v1/workspaces/{workspaceSlug}` |
| Invitations | `GET /api/v1/invitations/{token}`, `POST /api/v1/invitations/{token}/accept` |

The OpenAPI document of your project is the exact list. It contains only the routes your selection generated.

## Authentication

The session row is the credential. Every sign-in endpoint returns a token in the response body and sets no cookie:

```json
{"token":"...","expires_at":"...","user":{...}}
```

Send it on every request:

```text
Authorization: Bearer <token>
```

The token lives as long as `SESSION_EXPIRY`. Logout revokes it. Changing, setting, or removing the password revokes your other sessions and every [personal API token](/docs/features/api-tokens#when-tokens-are-revoked), same as on the web. A forgot password sign-in and the admin kill switch revoke every session and token.

A browser frontend on the same site may use the `session_token` cookie instead. Cookie requests that change state need the `X-CSRF-Token` header. Get the token from `GET /api/v1/csrf`. Requests with an `Authorization: Bearer` header never read cookies and need no CSRF token. Other schemes, such as Basic auth added by a proxy, keep the cookie and its CSRF check. To sign in with a cookie, see [Cookie sessions](#cookie-sessions).

Scripts, integrations, and AI agents use a [personal API token](/docs/features/api-tokens) instead of a session. A token works only on the routes opened to tokens, with a `read` or `write` scope.

Impersonation exists only in the web Admin panel and is cookie only. An impersonation session sent as a bearer token is not accepted. During impersonation, API reads work. Writes and the billing portal answer 403 `impersonation_read_only`.

### Cookie sessions

Browser apps should not hold a session token. Add `?session=cookie` to `auth/password`, `auth/magic-link/verify`, `auth/forgot-password/verify`, or `auth/email-change/verify`. The server sets the HttpOnly `session_token` cookie and leaves the token out of the body. Every API build has it. The [SvelteKit app](/docs/guides/sveltekit) signs in this way.

A cookie sign-in always needs the CSRF header, even before a session exists. Without it:

```bash
CSRF=$(curl -s -c jar -b jar http://localhost:8090/api/v1/csrf | jq -r .token)

curl -si -c jar -b jar -X POST 'http://localhost:8090/api/v1/auth/password?session=cookie' \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}' | grep -E '^(HTTP|Set-Cookie|\{)'
```

```text
HTTP/1.1 403 Forbidden
{"error":{"code":"forbidden","message":"Send the X-CSRF-Token header with cookie requests."}}
```

With it:

```bash
curl -si -c jar -b jar -X POST 'http://localhost:8090/api/v1/auth/password?session=cookie' \
  -H 'Content-Type: application/json' \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}' | grep -E '^(HTTP|Set-Cookie|\{)'
```

```text
HTTP/1.1 200 OK
Set-Cookie: session_token=b097807bc6e12df2d3b9bb0348dd7c46f4d5a3a48be59f28f642458b46b411ff; Path=/; Expires=Mon, 12 Oct 2026 22:01:43 GMT; HttpOnly; SameSite=Lax
{"expires_at":"2026-10-12T22:01:43.909830346Z","user":{"id":"5b8cac5a-7642-4afd-85fb-f4bb9904479f","email":"ada@example.com","name":"Ada Lovelace","password_set":true,"created_at":"2026-10-05T22:01:37.04082295Z"}}
```

The body has no `token`. In production the cookie is also `Secure`. When the browser already sent a `session_token` cookie, that session is revoked, same as a web sign-in.

From then on the cookie authenticates. Send `X-CSRF-Token` with every request that changes data. `POST /api/v1/auth/logout` revokes the session and clears the cookie. Requests without `session=cookie` keep the bearer behavior.

### Onboarding

A new account has no name yet. `GET /api/v1/account` reports `onboarding_required`. Every other authenticated route except `POST /api/v1/auth/onboarding` and `POST /api/v1/auth/logout` answers 403 `onboarding_required` until the user sends a name. Same rule as the web app.

With Workspaces, onboarding creates a default workspace. A person who signed up through an invitation should not get one. Send the token from the invitation link as `invitation_token`:

```bash
curl -s -X POST http://localhost:8090/api/v1/auth/onboarding \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Grace Hopper","invitation_token":"lsdmIvutTZ_h3LhaVc8D7xj34b3stJUkktP9GKrC8NQ"}' | jq .
```

```json
{
  "user": {
    "id": "dfbb778e-8d17-4e8f-9739-84eeebe3f7e6",
    "email": "grace@example.com",
    "name": "Grace Hopper",
    "password_set": false,
    "created_at": "2026-10-05T21:53:49.330794686Z"
  },
  "workspace": null
}
```

`workspace` is `null` because none was created. Then `POST /api/v1/invitations/{token}/accept` joins the invited workspace. A token that is unknown, used, revoked, expired, or meant for another address is ignored, and the default workspace is created as usual.

In a SvelteKit project with Workspaces, the invitation can also ride a cookie. `GET /api/v1/invitations/{token}` sets the HttpOnly `pending_invitation` cookie for a guest or an account without a name:

```text
Set-Cookie: pending_invitation=QGUYBZKDOBITWNcXxwi4rrMNsRP1nh8Xgnx-XWgagDo; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax
```

The app cannot read an HttpOnly cookie, so `GET /api/v1/account` reports it:

```json
{
  "user": {
    "id": "97c6cfe7-5439-4a98-adbb-7eee4ab6004c",
    "email": "linus@example.com",
    "name": "",
    "password_set": false,
    "created_at": "2026-10-05T21:53:58.09547733Z"
  },
  "onboarding_required": true,
  "workspaces": [],
  "is_admin": false,
  "pending_invitation": "QGUYBZKDOBITWNcXxwi4rrMNsRP1nh8Xgnx-XWgagDo"
}
```

Onboarding without `invitation_token` falls back to the cookie and answers `workspace: null`. A successful accept clears the cookie, and `pending_invitation` goes back to `null`. That works across tabs: the invitation opened in one tab, the magic link in another.

## Requests and responses

- Request bodies are `application/json`, at most 1 MiB. Other content types get 415 `unsupported_media_type`. Unknown fields get 400 `invalid_body`.
- Responses use snake case. Times are RFC 3339 in UTC.
- Lists return `{"items":[...]}`.
- Deletions and revocations return 204.
- Workspace activity is cursor paginated: `limit` (default 50, max 200) and `cursor`. The answer carries `next_cursor`.

## Errors

Every non-2xx answer uses one envelope:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Check the highlighted fields and try again.",
    "fields": {
      "name": "Name is required."
    }
  }
}
```

`fields` appears only for validation errors. Codes are lowercase snake case and stable. Show `message` to people. Branch on `code` in code.

| Status | Codes |
| --- | --- |
| 400 | `bad_request`, `invalid_body`, `invalid_token`, `invalid_cursor` |
| 401 | `unauthenticated`, `invalid_credentials` |
| 403 | `forbidden`, `insufficient_scope`, `onboarding_required`, `impersonation_read_only`, `invitation_email_mismatch`, `email_not_verified`, `feature_unavailable`, `limit_reached`, `quota_exceeded` |
| 404 | `not_found` |
| 405 | `method_not_allowed` |
| 409 | State conflicts such as `last_owner`, `seat_limit`, `checkout_conflict`, `password_already_set`, `token_limit`, `email_taken`, `already_member`, `active_subscription` |
| 413 | `body_too_large` |
| 410 | `invitation_unavailable` |
| 415 | `unsupported_media_type` |
| 422 | `validation_failed`, `invalid_plan`, `invalid_interval` |
| 429 | `rate_limited` with a `Retry-After` header |
| 500 | `internal` |

## Walkthrough

These commands ran against a fresh Headless project with Stripe, SQLite, and SMTP. Your tokens and IDs will differ. Stripe was replaced by a local test double, so the checkout URL is a placeholder. With real test keys you get a real Checkout link.

Request a magic link. An unknown address gets a new account, like the web flow:

```bash
curl -si -X POST http://localhost:8090/api/v1/auth/magic-link \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com"}' | head -1
```

```text
HTTP/1.1 202 Accepted
```

The email links to `APP_URL/auth/magic-link/<token>`. Send that token to the API:

```bash
curl -s -X POST http://localhost:8090/api/v1/auth/magic-link/verify \
  -H 'Content-Type: application/json' \
  -d '{"token":"90856cb5bd7f0ae56515e22fe71b421baf6cdd6b8d956a9eb8281f062ba6d827"}' | jq .
```

```json
{
  "token": "851af9ad7b272e006a76c06fe6a977d7f25c7988ac778ca157e62d6cd7034ae5",
  "expires_at": "2026-10-12T10:34:31.391724139Z",
  "user": {
    "id": "5c8de3e8-595c-4c01-91a7-2770ffba111d",
    "email": "ada@example.com",
    "name": "",
    "password_set": false,
    "created_at": "2026-10-05T10:34:30.80236309Z"
  }
}
```

Finish onboarding and set a password:

```bash
TOKEN=851af9ad7b272e006a76c06fe6a977d7f25c7988ac778ca157e62d6cd7034ae5

curl -s -X POST http://localhost:8090/api/v1/auth/onboarding \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace"}' | jq .

curl -si -X POST http://localhost:8090/api/v1/account/password/set \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"password":"correct-horse-battery-staple"}' | head -1
```

```text
{
  "user": {
    "id": "5c8de3e8-595c-4c01-91a7-2770ffba111d",
    "email": "ada@example.com",
    "name": "Ada Lovelace",
    "password_set": false,
    "created_at": "2026-10-05T10:34:30.80236309Z"
  }
}
HTTP/1.1 204 No Content
```

### Log in

```bash
curl -s -X POST http://localhost:8090/api/v1/auth/password \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}' | jq .
```

```json
{
  "token": "b91c27083366e32cd8972a81bf107c5a5c8a85e81691f4605ed643bc7ea02ce9",
  "expires_at": "2026-10-12T10:34:31.655678287Z",
  "user": {
    "id": "5c8de3e8-595c-4c01-91a7-2770ffba111d",
    "email": "ada@example.com",
    "name": "Ada Lovelace",
    "password_set": true,
    "created_at": "2026-10-05T10:34:30.80236309Z"
  }
}
```

A wrong password answers 401:

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "Invalid email or password."
  }
}
```

### Read the account

```bash
TOKEN=b91c27083366e32cd8972a81bf107c5a5c8a85e81691f4605ed643bc7ea02ce9

curl -s http://localhost:8090/api/v1/account \
  -H "Authorization: Bearer $TOKEN" | jq .
```

```json
{
  "user": {
    "id": "5c8de3e8-595c-4c01-91a7-2770ffba111d",
    "email": "ada@example.com",
    "name": "Ada Lovelace",
    "password_set": true,
    "created_at": "2026-10-05T10:34:30.80236309Z"
  },
  "onboarding_required": false,
  "subscription": {
    "id": "a7e9c913-5c54-4157-ab70-54a011a7335e",
    "plan_id": "free",
    "status": "active",
    "provider": "",
    "current_period_end": null,
    "amount": null,
    "currency": "",
    "interval": null,
    "created_at": "2026-10-05T10:34:30.802600492Z",
    "updated_at": "2026-10-05T10:34:30.802600492Z"
  },
  "access": {
    "plan_id": "free",
    "state": "free",
    "trial_ends_at": null,
    "access_ends_at": null
  },
  "is_admin": false,
  "avatar_url": null
}
```

`is_admin` is `true` only for a session of a verified address in `ADMIN_EMAILS`, and `false` for API tokens and during impersonation. Use it to show Admin links. The admin routes still check on every request. `avatar_url` exists with storage. With Workspaces, the account lists `workspaces` instead of `subscription` and `access`, because a subscription belongs to a workspace. A SvelteKit project with Workspaces also answers `pending_invitation`, see [Onboarding](#onboarding).

### Project name and support address

`GET /api/v1/meta` needs no session. A personal API token with `read` works too:

```bash
curl -s http://localhost:8090/api/v1/meta
```

```json
{"name":"Headless Stripe","support_email":"hello@example.com"}
```

`name` comes from `internal/brand`, `support_email` from `SUPPORT_EMAIL`.

### Start a checkout

```bash
curl -s -X POST http://localhost:8090/api/v1/billing/checkout \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"plan_id":"pro","interval":"monthly"}' | jq .
```

```json
{
  "url": "https://checkout.stripe.com/c/pay/cs_test_1"
}
```

Open that URL in a browser. The payment webhook updates the subscription like it does for the web app. `GET /api/v1/billing` lists the purchasable plans with their prices. A plan that cannot be bought answers 422 `invalid_plan`.

With Workspaces, the same routes live under `/api/v1/workspaces/{workspaceSlug}/billing` and need the billing permission of that workspace.

### Log out

```bash
curl -si -X POST http://localhost:8090/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN" | head -1
```

```text
HTTP/1.1 204 No Content
```

The token is gone. The next call answers 401 `unauthenticated`.

## Verification links

Magic link, password reset, email change, and invitation emails always link to `APP_URL`:

```text
APP_URL/auth/magic-link/<token>
APP_URL/auth/forgot-password/<token>
APP_URL/auth/verify-email-change/<token>
APP_URL/workspaces/invitations/<token>
```

In an htmx or Datastar project those are the existing web pages. In a SvelteKit project they are pages of the app, and `APP_URL` is the Go server that serves it. In a Headless project `APP_URL` is your frontend. Your page reads the token from the URL and calls the matching endpoint: `magic-link/verify`, `forgot-password/verify`, `email-change/verify` under `/api/v1/auth`, or `GET /api/v1/invitations/{token}` and `POST /api/v1/invitations/{token}/accept`.

## OAuth

Google and GitHub stay browser redirect flows. They start at `API_URL/auth/{provider}`, return to `API_URL/auth/{provider}/callback`, and end with a `session_token` cookie. The web app then lands on `/app`. A Headless project redirects to `APP_URL/app`, where your frontend reads `onboarding_required` from `GET /api/v1/account`. In a SvelteKit project the app and the Go server share one origin, so the OAuth buttons are plain links with `data-sveltekit-reload`, and `task dev` forwards `/auth/google` and `/auth/github` to Go.

`API_URL` is the public origin of the Go server. It exists only when OAuth is selected and defaults to `APP_URL`. When they differ, both must share a site, for example `app.example.com` and `api.example.com`, because the result is a cookie session.

Register `{API_URL}/auth/{provider}/callback` with the provider.

## CORS

Browser frontends on another origin need `CORS_ALLOWED_ORIGINS`:

```env
CORS_ALLOWED_ORIGINS=https://app.example.com,http://localhost:5173
```

Exact origins only, comma separated. Wildcards and paths fail at startup. Listed origins may call `/api/` with credentials and read `Retry-After` on a 429. Unlisted origins get no CORS headers. Keep it empty when the frontend is served from `APP_URL`, as in a SvelteKit project, or the clients are native apps.

## OpenAPI

Every project with the JSON API ships a hand-written OpenAPI 3.1 document at `internal/api/openapi.yaml`. The server serves it at `GET /api/v1/openapi.yaml`.

All `/api/v1` routes are registered from one table in `internal/api/routes.go`. A test compares that table with the document in both directions. Add a route without documenting it and `go test` fails.

## Rate limits

The API shares the web limiters. Sign-in and authentication emails count per visitor IP and per recipient. Email changes and workspace invitations count per user or workspace. A blocked request answers 429 `rate_limited` with `Retry-After`.

Behind a reverse proxy, set `TRUSTED_PROXY_CIDRS`. See [Configuration](/docs/configuration#trusted-reverse-proxies).
