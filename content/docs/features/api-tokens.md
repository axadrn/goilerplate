---
title: "API tokens"
description: "Personal API tokens let scripts, integrations, and AI agents act for a user with a read or write scope."
order: 2
---

## TLDR

Every project with the [JSON API](/docs/features/json-api) ships personal API tokens. A signed-in user creates one, copies it once, and hands it to a script, an integration, or an AI agent through the [MCP server](/docs/features/mcp).

```text
Authorization: Bearer pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

A token acts for its user, but only on the routes opened to tokens and only with its scope. It never signs in, never manages sessions or other tokens, and never touches email, password, billing, admin, or deletions.

There is no extra flag: `--api`, `--framework svelte`, or `--headless` is enough.

## Sessions or tokens

| | Session token | Personal API token |
| --- | --- | --- |
| Comes from | A sign-in endpoint | Settings or `POST /api/v1/account/tokens` |
| Looks like | 64 hex characters | `pat_` and 52 base32 characters |
| Lives | Until `SESSION_EXPIRY` or logout | Until its expiry or until it is revoked |
| Reaches | Every route the user may use | Routes opened to tokens, with its scope |
| Works on `/mcp` | No | Yes |

Use sessions for your own frontend and mobile app. Use tokens for everything that runs without a person at the keyboard.

## Create a token

In an htmx, Datastar, or SvelteKit project, users open Settings. The API Tokens card sits under the sessions card. They pick a name, the access (Read only, or Read and write), an expiry from 7 days to no expiration, and in a workspaces project the workspace. The token is shown once in a dialog with a copy button. Closing the dialog removes it from the page.

Every project also has the API, for session callers only:

```bash
curl -s -X POST http://localhost:8090/api/v1/account/tokens \
  -H "Authorization: Bearer $SESSION" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Claude","scopes":["read","write"],"expires_in_days":30}' | jq .
```

```json
{
  "token": "pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "api_token": {
    "id": "73ffd8b3-eaed-48ef-8d8a-3a2db5ea8d29",
    "name": "Claude",
    "prefix": "q7xpgvfa",
    "scopes": [
      "read",
      "write"
    ],
    "last_used_at": null,
    "expires_at": "2026-11-04T13:09:07.014635707Z",
    "expired": false,
    "created_at": "2026-10-05T13:09:07.014635707Z"
  }
}
```

| Field | Rule |
| --- | --- |
| `name` | Required, a label for the user. |
| `scopes` | `["read"]` by default. `write` implies `read`. |
| `expires_in_days` | 1 to 365, or `null` for a token that never expires. |
| `workspace_id` | Workspaces projects only. Required, a workspace the user belongs to. |

`token` appears only in this answer. The database keeps a SHA-256 hash, like sessions, and `prefix` so people can tell their tokens apart. An account holds at most 50 tokens, then creation answers 409 `token_limit`.

`GET /api/v1/account/tokens` lists them newest first, without the token or its hash, with `last_used_at` updated at most once a minute. Expired tokens stay listed with `expired: true` until they are revoked. `DELETE /api/v1/account/tokens/{id}` revokes one with 204.

## When tokens are revoked

Besides `DELETE /api/v1/account/tokens/{id}` and the Revoke button in Settings, these events revoke every token of the account at once, on the web and in the API:

- Signing in with a forgot password link.
- Changing, setting, or removing the password.
- An admin signing the user out everywhere from the Admin panel or `POST /api/v1/admin/users/{userID}/sessions/revoke`.

Sign out everywhere in Settings and `DELETE /api/v1/account/sessions` end sessions only. Tokens keep working until they expire or are revoked.

## Scopes

A `read` token may call reads. A token with `read` and `write` may also make the changes opened to tokens. Every other route refuses tokens with 403 `forbidden`, whatever the scope.

| Access | Routes |
| --- | --- |
| Read | `GET /api/v1/account`, `GET /api/v1/meta`, `GET /api/v1/openapi.yaml`, `GET /api/v1/csrf`. With workspaces: `GET /api/v1/workspaces`, `GET .../{workspaceSlug}`, `GET .../activity`, `GET .../billing`, `GET /api/v1/invitations/{token}`. Without workspaces: `GET /api/v1/billing`. |
| Write | `PATCH /api/v1/account/name`. With storage: `POST` and `DELETE /api/v1/account/avatar`. With workspaces: `PATCH .../name`, `POST .../invitations`, `POST .../invitations/{invitationID}/resend`, `DELETE .../invitations/{invitationID}`. |
| Never | Sign-in, onboarding and logout, email, password, sessions, API tokens, account deletion, creating, deleting or leaving workspaces, accepting invitations, member role changes, member removal, checkout, billing portal and management, admin. |

The routes table in `internal/api/routes.go` declares this per route with `TokenRead`, `TokenWrite`, or the default `TokenDenied`, so a new route stays closed to tokens until you open it. The OpenAPI document lists `apiTokenAuth` with the scope on every operation a token may call, and a test keeps the two in sync.

A read token on a write route:

```bash
curl -s -X PATCH http://localhost:8090/api/v1/account/name \
  -H "Authorization: Bearer $READ_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada King"}'
```

```json
{"error":{"code":"insufficient_scope","message":"This API token needs the write scope."}}
```

Any token on a route closed to tokens:

```bash
curl -s http://localhost:8090/api/v1/account/sessions \
  -H "Authorization: Bearer $TOKEN"
```

```json
{"error":{"code":"forbidden","message":"API tokens cannot use this endpoint. Sign in to continue."}}
```

## Workspaces

With workspaces, a token is bound to one workspace at creation. Every other workspace answers 404 `not_found`, even one the user belongs to, and lists show only the bound workspace. Roles still apply: a member's token cannot do what the member cannot do. When the user leaves the workspace, the token reaches nothing there.

People stay managed by people. A write token may invite members and manage invitations, but it never changes a role or removes a member. These ran against a Headless workspaces project with the owner's write token and then the owner's session:

```bash
curl -s -X PATCH http://localhost:8090/api/v1/workspaces/$SLUG/members/$MEMBER/role \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"role":"admin"}'
```

```json
{"error":{"code":"forbidden","message":"API tokens cannot use this endpoint. Sign in to continue."}}
```

The same request with the session answers 200 with the member and the new role. `DELETE .../members/$MEMBER` answers 403 `forbidden` to the token and 204 to the session.

## Revocation and expiry

```bash
curl -si -X DELETE http://localhost:8090/api/v1/account/tokens/73ffd8b3-eaed-48ef-8d8a-3a2db5ea8d29 \
  -H "Authorization: Bearer $SESSION" | head -1
```

```text
HTTP/1.1 204 No Content
```

The revoked token now gets 401:

```json
{"error":{"code":"unauthenticated","message":"Sign in to continue."}}
```

Expired and unknown tokens get the same answer. Revocation deletes the row, so nothing is left to reuse.

## Where tokens work

Tokens authenticate only `/api/` and `/mcp`. On a web page a token is ignored and the request is a guest, so it cannot open settings or the Admin panel. Requests with an `Authorization: Bearer` header never read cookies and need no CSRF token.
