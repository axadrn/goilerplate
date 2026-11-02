---
title: "SvelteKit"
description: "A SvelteKit app on the JSON API, embedded in the Go binary. Dev loop, build, deploy, sessions, CSP, and new pages."
order: 3
---

## TLDR

```bash
goilerplate new --module example.com/acme --framework svelte ./acme
cd acme
cp .env.example .env
task dev
```

Open the Vite URL that `task dev` prints. In production the Go binary serves the built app and the JSON API on one origin. Node.js is only needed to build.

## When to pick it

I recommend htmx or Datastar. One language, one render path, no build step for the UI. SvelteKit is for teams that already know it and want to keep it.

It is not a second backend. The app is a static single-page app in `web/`. It talks only to the [JSON API](/docs/features/json-api) of the same project. There is no server-side rendering, no Node.js server, no CORS, and no token in the browser.

`--framework svelte` implies `--api`. It works with every database, payment, email, workspace, OAuth, and storage choice. `--mcp` works too. Blog and docs are not available, because both are server-rendered templ pages.

The app covers what the htmx and Datastar apps cover for your selection: sign-in by magic link and password, password recovery, email changes, onboarding, the dashboard, settings with sessions and personal API tokens, billing, Admin, and with Workspaces, workspaces, invitations, and activity. It has no landing page, legal pages, analytics, or impersonation. `GET /` redirects to `/app`.

## Requirements

- Go and [Task](https://taskfile.dev), like every project.
- Node.js 22.17 or newer. 24 LTS is what the Dockerfile and CI use.
- pnpm 12.9.1. `npm install --global pnpm@12.9.1` works.

`goilerplate doctor` checks Node.js and pnpm in a SvelteKit project.

## Stack

| Part | Choice |
| --- | --- |
| Framework | SvelteKit 3 with `adapter-static`, Svelte 5 runes |
| Language | TypeScript |
| Styles | Tailwind CSS 4, the same theme tokens and fonts as the templ apps |
| Components | shadcn-svelte 1.7.0 on bits-ui, lucide icons |
| Charts | layerchart |
| Checks | `svelte-check` with zero errors and zero warnings |

Every version in `web/package.json` is pinned exactly. `pnpm-lock.yaml` is committed.

```text
web/
  embed.go              embeds web/build into the Go binary
  package.json          exact pins and the pnpm version
  vite.config.ts        Kit options and the dev proxy
  src/app.css           theme tokens
  src/lib/api.ts        the only place that calls fetch
  src/lib/types.ts      response types, mirrored from openapi.yaml
  src/lib/config.ts     brand name and module flags
  src/lib/components/   ui/ from shadcn-svelte, app/, auth/, settings/, and module folders
  src/routes/           (auth), (app), and admin
  static/               favicon, fonts, theme.js
internal/routes/spa.go  serves the app with its Content Security Policy
```

## Dev loop

```bash
task dev
```

```text
Local: http://localhost:5173 (Go server: localhost:8090)
```

The first run installs the web dependencies. Then it starts the Go server and the Vite dev server. Open the Vite URL. Vite forwards `/api`, `/mcp`, `/webhooks`, `/healthz`, `/auth/google`, and `/auth/github` to Go, so the browser sees one origin, like in production. Svelte changes reload in place.

`task dev` sets `APP_URL` to the Vite origin, so magic links, password resets, and OAuth callbacks come back to the dev server. Pick the ports with `task dev APP_PORT=8091 DEV_PORT=5174`. Restart `task dev` after Go changes.

Check the app:

```bash
task check
```

```text
COMPLETED 1202 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS
```

## Build

```bash
task build
```

That runs `pnpm install --frozen-lockfile` and `pnpm build` in `web/`, then `go build -o bin/server ./cmd/server`. `bin/server` contains the app.

`go build`, `go vet`, and `go test` work without Node.js. `web/build/.gitkeep` keeps the embed compiling. A server without a built app answers pages with `The web app is not built. Run task build or task dev.` and logs a warning at startup. The API still works.

Never commit `web/build`, `web/node_modules`, or `web/.svelte-kit`. The generated `.gitignore` keeps them out.

## Deploy

```bash
docker build -t acme-app .
docker run --env-file .env -p 8090:8090 acme-app
```

The Dockerfile builds the app in a `node:24-alpine` stage, builds the Go binary with the app embedded, and copies only that binary into the runtime image. A SvelteKit image on SQLite is about 53 MB and reports healthy through `/healthz`. There is no Node.js at runtime.

Everything in [Deployment](/docs/guides/deployment) applies. `APP_URL` is the public URL of this server. It serves the app and the API, and emails link to its pages.

The generated GitHub Actions workflow sets up Node.js 24 and pnpm, runs install, check, and build in `web/`, then the usual Go steps.

## Sessions

The app signs in with `?session=cookie` on the sign-in endpoints. The server sets the HttpOnly `session_token` cookie and answers without a token:

```json
{"expires_at":"...","user":{...}}
```

JavaScript never sees a session token. Nothing goes into `localStorage` or `sessionStorage`. Cookie sign-in and every other cookie request that changes data needs the `X-CSRF-Token` header with the token from `GET /api/v1/csrf`. `web/src/lib/api.ts` does that for you. It fetches the CSRF token once, retries once after a 403, sends JSON or `FormData`, and turns the error envelope into an `ApiError` with `status`, `code`, `message`, and `fields`.

The app reads `is_admin` from `GET /api/v1/account` to show Admin, and `GET /api/v1/meta` for the support address in the sidebar. With workspaces, an invitation link opened before sign-in sets the HttpOnly `pending_invitation` cookie, and the account reports it as `pending_invitation`. After sign-in and onboarding, `/app` sends the person back to the invitation. The details are in [JSON API](/docs/features/json-api#onboarding).

Logout is a full page load to `/auth`, so no account data stays in memory.

## Content Security Policy

`internal/routes/spa.go` serves the app with its own policy. A build without storage answers:

```text
default-src 'self'; script-src 'self' 'sha256-q9KYlevF7MAey/lnYjVa2KVGEu+HpsmBYMuFT/nQmUQ='; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
```

The hash belongs to the inline script SvelteKit writes into `index.html`. The server computes it at startup from the embedded file, so a new build never needs a config change. Your hash will differ. With storage, `img-src` adds `S3_ENDPOINT`.

Do not add inline scripts. Put code in `web/src` or a file in `web/static`. The theme switcher is `web/static/theme.js` for that reason. API answers keep the strict JSON policy of the API.

Other caching rules: page paths get `index.html` with `Cache-Control: no-cache`. Files under `/_app/immutable/` are cached for a year. A missing file with an extension is a 404, never the app.

## Add a page

A page is a folder under `web/src/routes`. Pages inside `(app)` get the sidebar layout, and its `+layout.ts` sends signed-out visitors to `/auth`. This one lists the sessions of the signed-in account at `/app/sessions`:

`web/src/routes/(app)/app/sessions/+page.ts`

```ts
import { api } from '#lib/api.js';
import type { AccountSession, List } from '#lib/types.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	await parent();
	const sessions = await api.get<List<AccountSession>>('/api/v1/account/sessions', { fetch });
	return { title: 'Sessions', sessions: sessions.items };
};
```

`web/src/routes/(app)/app/sessions/+page.svelte`

```svelte
<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatDayTime } from '#lib/format.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="space-y-6 p-4 sm:p-6">
	<h1 class="text-3xl font-bold tracking-tight">Sessions</h1>
	{#each data.sessions as session (session.id)}
		<Card.Root>
			<Card.Header>
				<Card.Title>{session.display_name}</Card.Title>
				<Card.Description>Last seen {formatDayTime(session.last_seen_at)}</Card.Description>
				{#if session.current}
					<Card.Action><Badge>This browser</Badge></Card.Action>
				{/if}
			</Card.Header>
		</Card.Root>
	{/each}
</div>
```

`title` becomes the document title, `Sessions - Acme`. Add the link to `web/src/lib/components/app/sidebar-navigation.svelte` next to Dashboard. `await parent()` waits until the layout has loaded the account. With workspaces, workspace pages live under `web/src/routes/(app)/app/[workspaceSlug]/`.

The page needs no Go change, because the route already exists in the API. A new endpoint goes into `internal/api` and `internal/api/openapi.yaml` first. Then add its response type to `web/src/lib/types.ts`.

## Add a component

UI primitives are shadcn-svelte components in `web/src/lib/components/ui`. Add one with the pinned CLI:

```bash
cd web
pnpm dlx shadcn-svelte@1.7.0 add progress
```

It asks once, then writes `src/lib/components/ui/progress`. Import it from `#lib/components/ui/progress/index.js`. Compose components from pages and `web/src/lib/components/app` through their props and `class`. Keep product behavior out of `components/ui`.

Run `task check` after every change. It must report zero errors and zero warnings.
