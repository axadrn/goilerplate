---
title: "Installation"
description: "Install goilerplate and run your first generated project"
order: 3
---

## TLDR

Install the CLI. Generate.

```bash
go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest
goilerplate new
```

That is it. No account, no login. The generated folder already contains a clean Git repository on `main` with an `Initial commit`.

## What you need

- Git 2.38 or newer.
- Go 1.26 or newer to install the CLI and build the generated project.
- Task for the short development commands.
- The Tailwind CSS standalone CLI for an htmx or Datastar project. `task dev` runs `tailwindcss`.
- Node.js 22.17 or newer and pnpm for a SvelteKit project.
- Docker only when you select PostgreSQL or want containerized local services.

## Install

Use this on macOS or Linux. On Windows, run the same Linux setup inside WSL. Native Windows is not supported. Neither is PowerShell.

```bash
go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest
```

The `/v4` part is required by Go modules. The installed command is still `goilerplate`. If the command is not found afterward, add `GOBIN` or `$(go env GOPATH)/bin` to your shell path.

You can also download a release binary from [GitHub Releases](https://github.com/axadrn/goilerplate/releases).

## Generate and run

```bash
goilerplate new
cd my-app
cp .env.example .env
# Configure the generated SMTP or Resend settings in .env.
task dev
```

`goilerplate new` renders the project on your machine. It needs no network. No `git init`, `git add`, or first commit is needed.

`task dev` starts the app at port `8090` and the templ proxy at `7331`. If either port is busy, it selects the next available port and prints the local URL. To set exact ports or open the browser automatically, run:

```bash
task dev APP_PORT=8091 DEV_PORT=7332 OPEN_BROWSER=true
```

The selected SMTP or Resend provider is used in every environment. The application reports all missing provider settings and stops at startup, so local development exercises the same delivery path as production.

Generated READMEs stay intentionally small. Use these public docs for setup, configuration, testing, and deployment.

## Something missing?

Run the read-only environment check inside the generated project:

```bash
goilerplate doctor
```

Every failed check includes the concrete fix.
