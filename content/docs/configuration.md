---
title: "Configuration"
description: "Typed environment configuration that validates only selected features"
order: 4
---

## TLDR

Copy the generated example and fill the values for your stack.

```bash
cp .env.example .env
task dev
```

The generated `.env.example` is the source of truth. It lists only the values this project can use.

The application reads process environment variables first, then fills missing values from `.env`. Git ignores `.env`. Production platforms should provide configuration through their environment or secret store.

## One rule

- Brand belongs in `internal/brand`.
- Architecture belongs in Go source.
- Deployment coordinates, external IDs, operator switches, and secrets belong in Env.

The generator writes the selected framework, database, mailer, payment provider, and modules into the project. Env configures that project. Env cannot switch it into another architecture.

## Application

```env
APP_ENV=development
APP_URL=http://localhost:8090
PORT=8090
SUPPORT_EMAIL=hello@example.com
```

`APP_ENV` selects runtime behavior. It does not select `.env.development` or `.env.production`.

Edit `internal/brand/brand.go` to change the product name, tagline, or description. Those values are reviewed source code, not deployment configuration.

### Development ports

`task dev` uses these defaults:

- Go application: `8090`
- templ proxy: `7331`

If a default is busy, goilerplate selects the next available port. The printed local URL points to the templ proxy, which forwards requests to the Go application and provides live reload.

Set exact ports or open the browser automatically when needed:

```bash
task dev APP_PORT=8091 DEV_PORT=7332 OPEN_BROWSER=true
```

The development task supplies the selected app port as `PORT` and the proxy URL as `APP_URL`. Port selection lives in `cmd/devport/main.go` and is wired through `Taskfile.yml`.

`APP_PORT`, `DEV_PORT`, and `OPEN_BROWSER` are development-task settings only. Production continues to use `PORT` and `APP_URL` from its environment.

### Development

Use `APP_ENV=development` only on localhost. It serves assets from disk without browser caching, permits localhost cookies without HTTPS, and defaults to debug logs. The selected email provider and its credentials are required in every environment.

### Production

Use `APP_ENV=production` behind HTTPS. It enables secure cookies, embedded cacheable assets, and info-level logs by default.

`APP_URL` must be the public origin. Emails, canonical URLs, and cookies depend on it. OAuth callbacks use `API_URL`, which defaults to `APP_URL` and exists only when OAuth is selected. In a SvelteKit project `APP_URL` is this server, which serves the app and the API. In a Headless project `APP_URL` is your frontend. The generated Dockerfile already sets `APP_ENV=production`; provide every other production value at runtime.

## Database

SQLite example:

```env
DB_CONNECTION=./data/app.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)
```

PostgreSQL example:

```env
DB_CONNECTION=postgres://user:password@localhost:5432/app?sslmode=disable
```

The driver is selected at generation time. There is no runtime database switch.

## Sessions and tokens

```env
SESSION_EXPIRY=168h
TOKEN_EMAIL_VERIFY_EXPIRY=24h
TOKEN_PASSWORD_RESET_EXPIRY=1h
TOKEN_EMAIL_CHANGE_EXPIRY=24h
TOKEN_MAGIC_LINK_EXPIRY=10m
```

Values use Go duration syntax such as `10m`, `24h`, and `168h`.

## Trusted reverse proxies

The included authentication rate limiter counts sign-in and authentication-email requests by visitor IP. Behind a reverse proxy, the direct connection belongs to the proxy. The app must trust that proxy before it can safely use the forwarded visitor IP.

Use this rule:

- App receives traffic directly: leave `TRUSTED_PROXY_CIDRS` empty.
- Proxy connects from loopback: leave it empty. Loopback is trusted automatically.
- Proxy connects over a private network: set the exact proxy network CIDR.

For a Docker network named `proxy`, get the value on the server:

```bash
docker network inspect proxy --format '{{(index .IPAM.Config 0).Subnet}}'
```

Then set that returned private network, for example:

```env
TRUSTED_PROXY_CIDRS=172.20.0.0/16
```

Trust only networks that connect directly to the app. Never use `0.0.0.0/0` or `::/0`. Otherwise anyone can forge a forwarded IP and bypass IP-based limits. Forwarded client headers from every untrusted address are ignored.

## Email

SMTP configuration:

```env
EMAIL_FROM=noreply@example.com
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_PASSWORD=...
SMTP_TLS=starttls
```

The sender must be valid for your provider. Username and password must either both be set or both be empty. Supported TLS modes are `starttls`, `tls`, and `none`.

Resend configuration:

```env
EMAIL_FROM=noreply@example.com
RESEND_API_KEY=re_xxx
RESEND_AUDIENCE_ID=
```

`RESEND_AUDIENCE_ID` is optional. Without it, newsletter signups remain local.

## Payments

The generated file lists only the selected provider. Its keys. Its webhook secret. Its plan identifiers. Keep test and live resources separate.

See [Subscriptions and Billing](/docs/features/subscriptions).

## Optional modules

Optional configuration appears only when the generated code can use it. This includes OAuth, S3 storage, Admin access, analytics, error tracking, and development seeds.

Projects with the JSON API add `CORS_ALLOWED_ORIGINS`. OAuth projects add `API_URL`. See [JSON API](/docs/features/json-api#cors).

## Validation behavior

Configuration loads into a typed struct and returns one combined error containing every invalid value.

Provider-specific secrets are validated only when that provider was selected. An SMTP project does not require a Resend key. A SQLite project does not validate PostgreSQL settings. Validation runs in every environment so the application never silently replaces the selected email provider.

## Render configuration

Request context contains `config.RenderConfig`, not the full application config. Templates can read only the public origin, support address, runtime environment, robots setting, and analytics IDs needed to render a response.

Secrets, database configuration, operator allowlists, provider credentials, and trusted proxy ranges stay in startup wiring. Adding a private setting to `Config` does not make it visible to templates.

## Secrets

Commit `.env.example`. Never commit `.env`.

Production secrets belong in the deployment platform's secret store. Rotate a secret immediately if it enters Git history.
