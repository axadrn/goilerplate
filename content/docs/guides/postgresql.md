---
title: "PostgreSQL Setup"
description: "Run the generated PostgreSQL project locally and in production"
order: 3
---

## TLDR

Select PostgreSQL during generation. Install Docker. Then run:

```bash
task dev
```

The generated Taskfile starts the included PostgreSQL service and waits until it is healthy.

Tests are separate from the development database. With Docker running, `go test ./...` creates and removes temporary PostgreSQL containers automatically. See [Testing](/docs/guides/testing).

## Local connection

The generated `.env.example` contains a development URL:

```env
DB_CONNECTION=postgres://goilerplate:dev_password@localhost:5432/goilerplate_dev?sslmode=disable
```

Copy the file to `.env`. Change credentials when the project leaves local development.

## Migrations

The app runs Goose migrations during startup before it serves traffic.

Migration files live in:

```text
internal/db/migrations/
```

Never rewrite an applied migration in a released product. Add the next numbered migration.

## Production

Use the connection URL from the managed database or private PostgreSQL service.

Enable TLS according to the provider. Keep the database on a private network when possible and grant the application user only the permissions it needs.

## Pooling

Start with the driver defaults. Measure connection use under real load before changing pool sizes.

Application replicas share PostgreSQL safely. Process-local rate limits still need a multi-replica design. So do background jobs.

## SQLite or PostgreSQL

SQLite is simpler for one writable application instance. PostgreSQL is better for multiple application replicas. It also fits managed database operations and PostgreSQL-specific features.

The generator chooses one driver. It does not ship both behind a runtime toggle.
