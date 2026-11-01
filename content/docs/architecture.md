---
title: "Architecture"
description: "Simple layers. Explicit ownership. Ordinary Go."
order: 5
---

## TLDR

Requests move in one direction:

```text
HTTP -> handler -> service -> repository -> database
                  |
                  -> model
HTTP <- templ response
```

Each layer has one job. There is no framework hiding the call path.

## Project shape

```text
cmd/server/             application entry point
internal/
  config/               typed environment configuration
  db/                   connection and migrations
  handler/              HTTP input and response choice
  middleware/           request boundaries
  model/                plain domain data
  repository/           explicit SQL
  routes/               route registration
  service/              business rules and transactions
  ui/                   templ layouts, pages, blocks, components
content/                optional Markdown blog and docs
assets/                 embedded CSS, JavaScript, fonts, and images
```

## Handler

A handler reads HTTP input. It calls one service operation. Then it selects the response. It does not contain SQL or billing rules.

## Service

A service owns business rules. It preserves the final workspace Owner. It replaces sessions after a credential change. It checks webhook state timestamps.

Transactions live here when one rule spans multiple writes.

## Repository

A repository owns SQL for one part of the domain. Queries require explicit tenant scope where applicable. There is no generic base repository.

## Model

Models are plain structs and small domain helpers. They do not know about HTTP. They do not know about templates or database connections.

## UI

Pages are templ components built from shadcn-templ. The frontend selection adds one small boundary. It owns attributes, partial responses, redirects, and the browser asset.

Handlers do not ask whether the project uses htmx 4.0 or Datastar 1.0.

## Two transports

With the JSON API, `internal/api` sits next to `internal/handler`. Both are thin. An HTML handler and its JSON twin read different input and write different output, then call the same service method. Rules, permission checks, and transactions exist once, in the service. A Headless project keeps `internal/api` and drops `internal/handler` and `internal/ui`. A SvelteKit project does the same and adds `web/`, a static app that calls only `/api/v1`. The Go binary embeds its build, and `internal/routes/spa.go` serves it. See [JSON API](/docs/features/json-api) and [SvelteKit](/docs/guides/sveltekit).

## Tenant scope

When Workspaces is selected, tenant data belongs to a workspace and every tenant query includes `workspace_id`. Middleware resolves the workspace and membership before a handler runs.

When Workspaces is not selected, the generated app has no workspace model. User-owned data stays simple.

## Why this scales

The code scales through:

- Clear ownership.
- Database constraints.
- Useful indexes.
- Atomic transactions.
- Explicit tenant scope.
- Tests around races and security boundaries.

It does not scale through generic repositories, a policy engine, an event bus, or dependency injection before a real need exists.
