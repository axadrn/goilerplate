# goilerplate generator

The generator and its master project tree live here. `_template/` mirrors the generated project layout.

A source is selected when every condition word in its file or directory path matches the configuration. The generator removes those words from the destination path. Files ending in `.tmpl` are rendered with `text/template`; all other files are copied.

| Axis | Words |
|---|---|
| `framework` | `htmx`, `datastar`, `headless`, `svelte` |
| `database` | `sqlite`, `postgres` |
| `payment` | `polar`, `stripe` |
| `mail` | `smtp`, `resend` |
| `workspaces` | `workspaces`, `no_workspaces` |
| `storage` | `storage`, `no_storage` |
| `google` | `google`, `no_google` |
| `github` | `github`, `no_github` |
| `blog` | `blog`, `no_blog` |
| `docs` | `docs`, `no_docs` |
| `api` | `api`, `no_api` |
| `mcp` | `mcp`, `no_mcp` |

For a boolean axis, the `no_` word selects its false value. A path carries only the words that change its selection, and a test enforces that minimum: a file word must change the selection of its file, a directory word the selection of at least one file below it. Words in one path element follow the table order above.

Several words of one axis in one path element are alternatives: `assets.htmx.datastar/` selects htmx or Datastar, so a Headless project never gets it. When an axis appears in several elements of a path, the configuration must match each of them: `internal/ui/layouts.htmx.datastar/base.htmx.templ` selects htmx only. Elements whose words of one axis exclude each other are an error.

Vendored sources keep the path their sync script writes and cannot carry words. `vendoredDirectories` in `select.go` gives them their words instead: `internal/ui/components/` belongs to htmx and Datastar.

`svelte` selects the SvelteKit frontend: the Headless backend plus the app in `web.svelte/`. Backend sources both share carry `headless.svelte`, such as `internal/api/webhook.headless.svelte.go`. SvelteKit has its own `Dockerfile.svelte` (a Node.js stage builds the app before Go embeds it), `.dockerignore.svelte` and `.github/workflows/ci.svelte.yml`. The JSON root `internal/api/root.headless.go` stays Headless only, and SvelteKit's own backend sources carry `svelte`, such as `internal/routes/spa.svelte.go`, which serves the embedded build from `web/embed.go`. File names under `web.svelte/` belong to the SvelteKit toolchain (`button.svelte`, `vite.config.ts`, `app.d.ts`, `session.svelte.ts`), so `literalFileNameDirectories` in `select.go` keeps them literal. There only directory elements carry words, as in `web.svelte/src/routes/(app)/app.workspaces/+page.ts`, and a final `.tmpl` still renders. Directory dots always introduce condition words, so the app has no `[...rest]` route and answers unknown pages with `+error.svelte`. `scripts/sync-shadcn-svelte.sh` vendors the shadcn-svelte components into `web.svelte/src/lib/components/ui/` and writes `package.json` and `pnpm-lock.yaml`; pnpm never runs inside `_template`.

An unknown middle segment is an error. Numeric segments and `min` are allowed as literal names. Alternatives are explicit: `account.htmx.go` and `account.datastar.go` both generate `account.go`. Two selected sources for one destination are a collision and an error.

Copied files still replace the literal tokens `{{.ModulePath}}`, `{{.ProjectNameGo}}`, and `{{.DatabaseTimestamp}}`.

To add a word, add one unique word to its axis in `select.go` and its configuration mapping. To add an axis, add its words and mapping, then use only the required words in source paths.

A fresh project carries the template's superset `go.sum`. The first `go mod tidy` removes checksums that its selected `go.mod` does not need.

## Configuration contract

The generated source owns the project identity and architecture:

- `internal/brand` owns the name, tagline, and description.
- Generated Go source fixes the framework, database, mailer, payment provider, and included modules.
- `.env.example` contains only deployment coordinates, external IDs, operator switches, and secrets used by that exact build.

Do not add Env switches for generator choices. If a project needs a different database or provider, generate that architecture or change the owned source explicitly.

Only `config.RenderConfig` enters request context. It contains the public values templates need. Full configuration, secrets, database settings, operator allowlists, and trusted proxy ranges stay in application wiring.

Run `go test ./...` from this directory.
