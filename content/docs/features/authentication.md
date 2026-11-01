---
title: "Authentication"
description: "Database sessions. Passwordless email. Passwords. OAuth. Safe account changes."
order: 1
---

## TLDR

Authentication uses revocable database sessions. Not JWTs.

Users sign in with the methods you select. Magic links. Passwords. Google. GitHub.

## Session model

The browser stores one random token in a secure cookie. The database stores only its hash.

Every authenticated request checks the hash. It checks the expiry and user too. Activity timestamps update at a throttled interval. The update is best-effort. A temporary database lock does not log out a valid user.

Logout deletes the server session before clearing the browser cookie.

## Magic links

1. The user enters an email address.
2. The app sends a short-lived single-use link.
3. The user opens it.
4. The server consumes the token and creates a new session.

Only a token hash is stored. Request responses do not reveal whether an email address exists.

## Passwords

Users may add a password. They may change or remove it. Password recovery replaces every active session for that user. Only one fresh session remains in the requesting browser.

Credential changes revoke other sessions. This removes copied or forgotten sessions after a sensitive account change.

## Email changes

A new email address is not stored as the account identity until its verification token is consumed.

Successful change revokes other sessions and creates a replacement current session. Rate limits apply per user and target address.

## OAuth

Google and GitHub live in isolated generated handlers. Select Google, GitHub, both, or neither. Env supplies client IDs and secrets only for the providers present in that build.

OAuth state is signed and short-lived. It is tied to the browser flow. Callback errors do not expose provider tokens.

OAuth signs in only with an address the provider has verified. Google must report `verified_email`. GitHub uses the public address, or the primary one when it is private, and `/user/emails` must list it as verified. Otherwise the sign-in fails and no account is created or linked.

Register `{API_URL}/auth/{provider}/callback` with each provider. `API_URL` defaults to `APP_URL`. Set it when the Go server has its own public origin, for example behind a Headless frontend.

## Rate limits and visitor IPs

goilerplate ships IP-based limits for sign-in, OAuth, magic-link, and password-recovery requests. Email delivery also has a separate limit for each normalized recipient. Account changes and workspace invitations use signed-in user or workspace keys instead of visitor IPs.

If the app runs behind a private-network reverse proxy, set `TRUSTED_PROXY_CIDRS` to that proxy network. This lets the limiter use the real visitor IP without trusting forged forwarded headers. Direct deployments and loopback proxies need no value.

See [Deployment](/docs/guides/deployment#reverse-proxy-trust) for the decision rule and Docker command.

## Onboarding

The profile name is validated before trimming and storage. Control characters are rejected so names cannot inject email or log headers.

With Workspaces selected, normal onboarding creates one workspace. It also creates an Owner membership. Invitation onboarding joins the invited workspace. It does not create an extra workspace.

## Session maintenance

Expired sessions are removed opportunistically with an indexed global cleanup. Account settings also show active sessions and allow revocation.

## Relevant code

```text
internal/handler/auth.go
internal/middleware/auth.go
internal/repository/session.go
internal/service/auth.go
internal/model/session.go
```
