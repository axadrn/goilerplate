---
title: "Introduction"
description: "What goilerplate is. What it generates. Where to start."
order: 1
---

## TLDR

goilerplate creates a production-ready Go SaaS from one command. It is free and open source.

You choose the stack. The CLI renders a normal Go repository on your machine. You own every file.

```bash
goilerplate new
```

That is the core product.

## What you get

Every project starts with the boring foundation already connected:

- Go server with graceful shutdown and typed configuration.
- shadcn-templ components rendered with templ, or shadcn-svelte in a SvelteKit app.
- htmx 4.0 or Datastar 1.0 for server-driven interactions, a SvelteKit app on the JSON API, or Headless with no native frontend.
- An optional [JSON API](/docs/features/json-api) under `/api/v1` on the same services.
- An optional [MCP server](/docs/features/mcp) at `/mcp`, so your users can connect their AI agents with a [personal API token](/docs/features/api-tokens).
- SQLite or PostgreSQL with versioned migrations.
- Authentication, sessions, security headers, validation, and rate limits.
- Optional payments, workspaces, OAuth, email, storage, blog, and docs.
- Stack-aware `AGENTS.md`, public `/llms.txt`, and raw Markdown documentation.
- Dockerfile. Taskfile. Health checks. Logs. A selection-aware Go test suite and GitHub Actions CI.

The generator includes only selected modules. Choosing Stripe does not leave Polar code behind. Leaving Workspaces disabled does not generate workspaces, memberships, or invitations.

### Frontend versions

goilerplate vendors and verifies the exact browser runtime selected during generation.

| Selection | Product line | Exact browser runtime | Server integration |
| --- | --- | --- | --- |
| htmx | htmx 4.0 | htmx 4.0.0 | Native htmx responses |
| Datastar | Datastar 1.0 | Datastar 1.0.3 | `datastar-go` 1.2.2 |
| SvelteKit | SvelteKit 3, Svelte 5 | Built by Vite, embedded in the binary | `/api/v1` JSON handlers |
| Headless | JSON API only | None | `/api/v1` JSON handlers |

Each generated project contains exactly one frontend. The runtime is served locally from a versioned, content-hashed path.

htmx and Datastar projects can add the JSON API with `--api`. They are what I recommend. SvelteKit always includes the API and is for teams that already use it: a static app in `web/`, built with Node.js and served by the Go binary on the same origin. See [SvelteKit](/docs/guides/sveltekit). Headless always includes the API and has no templ, Tailwind, or assets. Bring your own React, Svelte, or mobile client.

## What you do not get

goilerplate is not a framework or hosted application platform.

- No proprietary runtime.
- No account, no license and no server.
- No repository access.
- No source upload. Nothing leaves your machine.
- No hidden code generator inside your app.
- No requirement to keep the original architecture.

After generation, it is your Go repository.

## The simplest path

1. [Install the CLI](/docs/installation).
2. Open the [stack builder](/new) or run `goilerplate new`.
3. Copy `.env.example` to `.env`, configure email, then run `task dev`.
4. Read [Architecture](/docs/architecture) before changing boundaries.

## License

goilerplate is open source. The CLI is MIT. The template and every project it generates are 0BSD: use, change and sell what you build, with no notice to keep. Generated projects ship without a LICENSE file, so you choose your own.

Every frontend, both databases, both payment providers, both mail providers and every optional module are included. `goilerplate update` brings later template versions into your project.
