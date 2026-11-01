---
title: "Testing"
description: "Run the selected test suite with ordinary Go commands."
order: 5
---

## TLDR

Run the native Go test command:

```bash
go test ./...
```

That is the whole interface. There is no goilerplate test wrapper.

SQLite tests need nothing else. PostgreSQL tests require Docker to be running. They start isolated PostgreSQL 17 containers automatically and remove them when the package finishes.

The first PostgreSQL run may take longer while Docker downloads the image.

## What ships

The suite follows the generated selection. Provider and optional-feature tests ship only when relevant. Unselected implementations and their tests stay out.

| Selected foundation | Tests that ship |
| --- | --- |
| Every project | Configuration, migrations, authentication, sessions, email rendering, middleware, validation, and security boundaries |
| SQLite | Real migrations, repositories, transactions, constraints, and concurrent database behavior on SQLite |
| PostgreSQL | The same database suite on temporary PostgreSQL 17 containers, including PostgreSQL migrations, queries, transactions, and locks |
| Stripe | Selected factory, status mapping, invalid signatures, and webhook subscription behavior |
| Polar | Selected factory, status mapping, invalid signatures, and webhook subscription behavior |
| Workspaces | Membership scope, invitations, roles, tenant isolation, seat limits, billing ownership, and concurrency races |
| Google or GitHub OAuth | Selected provider state cookies and account provisioning |
| SMTP or Resend | Only the selected mail adapter behavior |
| S3-compatible storage | Bucket use, upload, deletion, URL signing, and file ownership |
| htmx 4.0, Datastar 1.0, SvelteKit, or Headless | Only the selected frontend: policy, handler, and UI behavior for htmx or Datastar, the app handler, CSP and route reservations for SvelteKit, the JSON root and JSON errors for Headless |
| JSON API | Every endpoint, bearer and cookie sessions, CSRF on cookie requests, CORS, personal API tokens and scopes, and a test that keeps the OpenAPI document and the routes in sync |

These tests protect the generated foundation. Add tests for the product behavior you build on top.

## PostgreSQL and Docker

Make sure Docker is available:

```bash
docker info
go test ./...
```

The test code chooses free local ports, starts temporary databases, and cleans them up. It does not use the development database from `compose.yml` and does not require a test connection string.

The Go command tests packages in parallel. A PostgreSQL project can therefore run several small isolated containers at once. This keeps packages independent and the code readable.

## Race detector

Run the same check as generated GitHub Actions:

```bash
go test -race ./...
```

The generated workflow creates templ and CSS assets, checks module files, builds, vets, and runs the full selected suite with the race detector.
