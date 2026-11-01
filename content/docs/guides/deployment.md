---
title: "Deployment"
description: "Build one container. Configure the environment. Run the generated Go app."
order: 1
---

## TLDR

The generated project includes a multi-stage Dockerfile. Build one image. Provide the environment. Run it.

```bash
docker build -t acme-app .
docker run --env-file .env -p 8090:8090 acme-app
```

The image already contains the selected architecture and brand. Env supplies only deployment coordinates, external IDs, operator switches, and secrets. Use the generated `.env.example` as the exact checklist.

Want it on your own server with HTTPS and SQLite backups in one command? [Deploy with ops](/docs/guides/deploy-with-ops).

## Recommended shape

Use one reverse proxy, one application container, and either:

- One persistent volume for SQLite.
- One PostgreSQL service.

Run this shape on an ordinary container host. Docker Compose. Coolify. Railway. Render. Fly.io.

## Required production values

At minimum, review:

```env
APP_ENV=production
APP_URL=https://app.example.com
SUPPORT_EMAIL=support@example.com
DB_CONNECTION=...
```

Then add the operator allowlist and secrets for the selected Admin, email, payment, OAuth, storage, analytics, and error features.

## Reverse proxy trust

The included authentication rate limiter counts sign-in and authentication-email requests by visitor IP. A reverse proxy hides that IP unless the app trusts the proxy's private network.

Decide once:

- No reverse proxy: leave `TRUSTED_PROXY_CIDRS` empty.
- Proxy connects from loopback: leave it empty. Loopback is already trusted.
- Proxy connects over a private network: set the exact CIDR of that network.

For a Docker network named `proxy`, run this on the server:

```bash
docker network inspect proxy --format '{{(index .IPAM.Config 0).Subnet}}'
```

Use the returned private subnet:

```env
TRUSTED_PROXY_CIDRS=172.20.0.0/16
```

Trust only a network that connects directly to the app. Never use `0.0.0.0/0` or `::/0`. Trusting the whole internet lets clients forge their IP and bypass IP-based limits. The app ignores forwarded client headers from every untrusted peer.

## SQLite

The database lives in `data/`, inside the container at `/app/data`. Keep that folder on a volume, or every new container starts with an empty database.

A SQLite project comes with a `compose.yml` that does this for you:

```yaml
services:
  web:
    build: .
    volumes:
      - ./data:/app/data
```

It runs the production build, so `docker compose up` needs the production env. For local work use `task dev`. With plain Docker, mount the folder yourself: `docker run --env-file .env -p 8090:8090 -v ./data:/app/data acme-app`. Run one writable application replica.

Back up the database file and verify restoration. WAL mode and busy timeout are already in the default connection string.

## PostgreSQL

Use a private production database URL with TLS. Multiple application replicas can share it, but process-local rate limits require a shared replacement before horizontal scaling.

## Health and shutdown

The server exposes its health endpoint and handles termination signals. The container receives time to stop accepting requests and finish active work before exit.

Configure the platform health check and graceful shutdown timeout around that behavior.

## Build once

Prefer an immutable image tag or digest. Build in CI. Scan and test the artifact. Deploy the exact image that passed.

Do not compile source independently on every production server.

## Final checklist

1. HTTPS and public `APP_URL` match.
2. Database persists and restore is tested.
3. Provider webhooks point to the public route.
4. OAuth callbacks match `API_URL`, or `APP_URL` when `API_URL` is empty.
5. Private proxy CIDRs match the real direct network, or remain empty when no value is needed.
6. Secrets exist only in the deployment store.
7. Health check and graceful shutdown work.
8. Logs and error reporting reach the chosen tools.
9. Operator addresses are set and signed up.
