---
title: "Generate a Project"
description: "Choose a stack and create it with one local command"
order: 1
---

## TLDR

Run the wizard:

```bash
goilerplate new
```

Or use the [website builder](/new). Copy its command. Paste it into your terminal.

Done. The new folder is already a clean Git repository on `main` with an `Initial commit`.

## ELI5

The website does not generate a project. It builds a command.

The template is built into the CLI. The CLI renders it with your choices on your machine, puts it in a new folder, starts Git, and commits it. No network is needed, and nothing leaves your machine.

## Interactive mode

```bash
goilerplate new
```

Use the arrow keys to move. Press Space to toggle choices. Press Enter to continue. The last screen shows the complete selection before generation.

## Explicit flags

Flags are better for scripts and repeatable setup:

```bash
goilerplate new \
  --name Acme \
  --module github.com/acme/product \
  --framework datastar \
  --database postgres \
  --payment stripe \
  --mail resend \
  --workspaces \
  --oauth google,github \
  ./acme
```

Interactive mode and flags use the same parser and generator path. The wizard is only a friendly way to assemble flags.

## Frontend, JSON API, and MCP

| Flag | Result |
| --- | --- |
| `--framework htmx` | htmx 4.0 web app. The default. |
| `--framework datastar` | Datastar 1.0 web app. |
| `--framework svelte` | [SvelteKit](/docs/guides/sveltekit) app on the JSON API, served by the Go binary. Implies `--api`. |
| `--api` | Adds the [JSON API](/docs/features/json-api) under `/api/v1` to an htmx or Datastar project. |
| `--headless` | The Go backend with the JSON API and no native frontend. |
| `--mcp` | Adds an [MCP server](/docs/features/mcp) at `/mcp`, so your users can connect their AI agents with a [personal API token](/docs/features/api-tokens). Needs the JSON API. |

`--headless` implies `--api` and cannot be combined with an explicit `--framework` or with `--content`. `--framework svelte` implies `--api` and cannot be combined with `--content` either. Blog and docs need a server-rendered frontend. Everything else stays selectable.

```bash
goilerplate new \
  --name Acme \
  --module github.com/acme/product \
  --headless \
  --database postgres \
  --payment stripe \
  --workspaces \
  ./acme
```

```bash
goilerplate new \
  --name Acme \
  --module github.com/acme/product \
  --framework svelte \
  --mcp \
  --database sqlite \
  --payment stripe \
  --mail smtp \
  --workspaces \
  --oauth github \
  ./acme
```

A SvelteKit project needs Node.js 22.17 or newer and pnpm to build. `goilerplate doctor` checks both.

The wizard asks for the frontend first and then, for htmx or Datastar, whether to include the JSON API. With the JSON API, with SvelteKit, or with Headless, it asks whether to include the MCP server. SvelteKit and Headless skip the JSON API question and the content step.

## The lock file

Every project contains `goilerplate.lock`. It records the template version and normalized generator config. The automatic `Initial commit` already includes it.

Keep it committed. `goilerplate update` uses it to recreate the old and new generated trees.
