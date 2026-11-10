# Changelog

What shipped in goilerplate, newest first.

## v4.0.2

November 10, 2026

**In short:** goilerplate needs no mail address any more. Support happens on GitHub.

### Changed
- **Commits from the CLI**: `goilerplate new` and `goilerplate update` commit as `goilerplate` with no email address.
- **Support on GitHub**: questions and bugs go to GitHub Issues. Security reports go through GitHub's private reporting.

### Upgrade
Install the new CLI: `go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest`.

## v4.0.1

November 4, 2026

**In short:** new projects deploy with moin, the new name of ops, so they keep working after the old name goes on December 4.

### Changed
- **compose.yml speaks moin**: new projects get an `x-moin` block instead of `x-ops`. Projects you already made keep working until December 4; rename `x-ops:` to `x-moin:` in their compose.yml before then.
- **Deploy with moin**: the guide moved to [Deploy with moin](https://goilerplate.com/docs/guides/deploy-with-moin); the old address leads there.

### Upgrade
Install the new CLI: `go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest`.

## v4.0.0

November 2, 2026

**In short:** goilerplate is free and open source: the CLI, the generator with every module, the docs and this website, in one public repository. No license, no account, no server.

### Why

A few days ago I ended the Free edition, because AI changed how people build apps: an agent writes login, billing and a dashboard in an afternoon. The numbers since then said the same thing. Selling a boilerplate no longer makes sense.

So I am following that thought to its end instead of going back and forth. goilerplate is free and open source, like shadcn-templ. I no longer sell code. I build ops, which deploys and backs up the apps goilerplate makes, on your own servers. Code is free, running it is the business.

If you bought goilerplate or templUI Pro: thank you. Everything you generated keeps working, and you get ops Hosted at half price for as long as you use it.

### Changed
- **`goilerplate new` works offline**: it renders the project on your machine, with no login and no network.
- **`goilerplate update`** downloads the template your project was made with from the Go module proxy and merges with Git on your machine, as before. It works for projects made with v4 or newer; projects made with v3 keep working, but do not update to v4.
- **Licenses**: the CLI is MIT. The template and every generated project are 0BSD, with no notice to keep. New projects ship without a LICENSE file, so you pick your own.
- **The module path ends in `/v4`.**

### Removed
- **`login`, `logout`, `whoami`, `claim`, `license` and `account`**: there is nothing to sign in to any more.

### Upgrade
Install the new CLI: `go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest`.

## v3.2.3

November 1, 2026

**In short:** `goilerplate update` says clearly when your Git is too old.

### Fixed
- **Git version check**: `goilerplate update` needs Git 2.38 or newer. With an older Git it printed only a cryptic `git merge-tree` usage line; now it names the required and the installed version before downloading anything.

### Upgrade
Update the CLI with `go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest` or the binary from the release, then run `goilerplate update`.

## v3.2.2

October 31, 2026

**In short:** Short workspace URLs, like `/app/acme/…`.

### New
- **Short workspace URLs**: a new workspace gets the slug of its name, `/app/acme/…` instead of `/app/acme-s-workspace-1a2b3c4d/…`. The workspace from sign-up takes the person's name, `/app/ada-lovelace/…`.
- **A number only when needed**: `acme-2` when the slug is taken. The unique index decides, so two workspaces created at the same moment never get the same one.
- **Plain letters**: accented letters become plain ones (`Ärzte Zürich` is `arzte-zurich`), and names of app pages such as `settings` or `admin` are reserved.

### Fixed
- **Workspace switcher**: it shows the first letter of a name that starts with a non-ASCII letter, such as `Ä`, instead of a broken character.
- **Docs table of contents**: it marks the last headings of short pages as you reach them.

### Upgrade
Commit your work, run `goilerplate update`, switch to the `goilerplate-update-v3.2.2` branch and run `go tool templ generate`. Existing workspaces keep their slug, so no link breaks; there is no migration.

## v3.2.1

October 30, 2026

**In short:** Two fixes for the new CLI wizard and its help.

### Fixed
- **Wizard focus**: `goilerplate new` without options lets you type the project directory again. The first step was not focused since v3.2.0.
- **Help**: `goilerplate help new` lists every option, starting with the required `--module`.

### Upgrade
`go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest`.

## v3.2.0

October 30, 2026

**In short:** One product, no Free edition. AI made a free starter pointless: anyone can generate the easy 80% in an afternoon. What takes weeks is the last 20%: billing that survives webhooks, tenant isolation, admin tools, and updates that keep coming.

### New
- **shadcn-templ v2.0.0** is stable.

### Changed
- **`goilerplate new` needs a license** and has no `--edition` flag. The wizard starts at the project directory and asks every module question; `--payment` is `stripe` or `polar`.
- **Every account from before today got the full license**, for free.
- **`goilerplate doctor`** shows only the template version.
- **`goilerplate new --help`** prints the help for `new`.
- **The price** goes from $99 to $149 on December 1, 2026. Every license bought before keeps lifetime updates.

### Removed
- **The Free edition.** Projects generated with it cannot run `goilerplate update`, because there is no Free template to update to: generate a new project with `goilerplate new` and move your code over.

### Upgrade
1. Install the CLI v3.2.0: `go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest`. Older CLIs default to the Free edition, which the server now refuses.
2. Commit your work, run `goilerplate update`, then switch to the `goilerplate-update-v3.2.0` branch.
3. Run `go tool templ generate` and `go mod tidy`, then build and test.

## v3.1.1

October 30, 2026

**In short:** A security fix: templ v0.3.1100, which needs Go 1.26.

### Fixed
- **templ v0.3.1100**: code generated by earlier versions does not sanitize spread attributes such as `{ attrs... }`, or URLs in attributes such as `<iframe src>`. Every vendored component passes `Attributes` through a spread, so every project with an htmx or Datastar frontend was affected. Headless and SvelteKit projects have no templ code.
- **goldmark v1.8.6** for docs and blog content (CVE-2026-5160).
- **Docker images on `alpine:3.24`** instead of `alpine:3.20.2`, which no longer gets security updates.
- **`golang.org/x/crypto` v0.57.0, `golang.org/x/net` v0.59.0** and the other `golang.org/x` modules templ pulls in.

### Changed
- **Go 1.26**: generated projects say `go 1.26.0` and build on `golang:1.26`. `goilerplate doctor` reads the version from your `go.mod` and asks for Go 1.26.
- **shadcn-templ v2.0.0-beta.16** in `AGENTS.md`. The vendored components are unchanged.

### Upgrade
1. Install Go 1.26 or newer.
2. Commit your work, run `goilerplate update`, then switch to the `goilerplate-update-v3.1.1` branch. By hand instead: `go get github.com/a-h/templ@v0.3.1100 github.com/yuin/goldmark@v1.8.6`, `go mod edit -go=1.26.0`, and `golang:1.26` and `alpine:3.24` in your `Dockerfile`.
3. Run `go tool templ generate` and `go mod tidy`, then build and test. Code generated by an older templ no longer compiles against v0.3.1100, so regenerating is required.

## v3.1.0

October 6, 2026

**In short:** One Go backend, every way to talk to it: HTML, a JSON API, MCP and SvelteKit. Services still own the rules; the new transports are thin and call the same methods as your htmx or Datastar handlers.

### New
- **JSON API (`--api`)**: `/api/v1` next to your htmx or Datastar app, from the same binary. Free covers auth, account and sessions; Paid adds billing, admin, workspaces, members, invitations, activity and avatars.
- **API details**: the login session doubles as the credential, `Authorization: Bearer <token>`; cookie sessions keep working with `X-CSRF-Token` from `GET /api/v1/csrf`. One error envelope, snake case JSON, cursor pagination for activity, 415 for anything that is not `application/json`, a 1 MiB body limit.
- **OpenAPI**: a hand-written OpenAPI 3.1 document at `/api/v1/openapi.yaml`, with a test that fails when a route and the document drift apart.
- **CORS** for listed origins through `CORS_ALLOWED_ORIGINS`; OAuth callbacks can use a separate `API_URL`.
- **Headless (`--headless`)**: the Go backend with the JSON API and no native frontend, no templ, Tailwind or assets. `GET /` answers JSON and the Docker image is about 51 MB.
- **Personal API tokens** (`pat_...`) with read or read and write scope, bound to one workspace, stored as hashes, shown once, created and revoked under Settings, Security. New routes are closed to tokens until they opt in; tokens never manage people, credentials or billing. They end when the password is reset or changed and when an admin revokes a user's sessions.
- **MCP server (`--mcp`)** at `/mcp`, built on the official Go SDK, so your customers connect Claude, Cursor or VS Code with a token. Free ships `whoami`; Paid adds `get_billing`; Workspaces add `list_workspaces`, `get_workspace`, `list_activity`, `invite_member` and `rename_workspace`. `AGENTS.md` shows how to add your own tool in under 20 lines.
- **SvelteKit (`--framework svelte`)**: a SvelteKit 3 app with Svelte 5 and shadcn-svelte on the Headless backend, embedded into the Go binary: one origin, one deploy, about 53 MB. Same look and pages as the templ app, cookie sign-in with CSRF, a strict CSP. Needs Node.js 22.17 or newer and pnpm; `goilerplate doctor` checks both. htmx and Datastar stay the recommendation.
- **The CLI** has `--api`, `--headless`, `--mcp` and `--framework svelte`, and the wizard asks for them.

### Changed
- **shadcn-templ v2.0.0-beta.13**: Base UI behavior across components, a minified bundle from `/assets/js` and gzip for text assets. The bundle goes from 613 KB to about 77 KB on the wire.
- **Safer by default**: OAuth refuses provider accounts without a verified email, payment webhooks reject bodies over 1 MiB, the invitation limit is shared across web, API and MCP, and invitation tokens no longer show up in request logs.

### Fixed
- **UI**: select triggers show their labels again, the mobile sidebar closes on the first Escape, and Back and Forward work after docs navigation in htmx and Datastar.
- **Datastar settings**: Sign Out Everywhere, Remove Password and remove avatar no longer stay disabled.
- **Forms** show their red border on errors.
- **Account deletion**: a refused deletion no longer deletes the user's files first.
- **A data race** in the first concurrent page renders is gone.

### Upgrade
1. Install the goilerplate CLI v3.1.0. The v3.0.x CLI can run this update, but it cannot read locks written by v3.1.0.
2. Commit your work, run `goilerplate update`, then switch to the `goilerplate-update-v3.1.0` branch.
3. If `internal/routes/routes.go` conflicts, keep the new `middleware.Gzip(assetHandler)` line and delete `mux.Handle("GET /components/{bundle}", components.ScriptsHandler())`. Vendored components you edited may also conflict.
4. If you edited component JavaScript or added components the template does not ship, re-add them with `go run github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@v2.0.0-beta.13 add <name> --overwrite`, run the same command with `bundle`, and commit `assets/js/shadcn-templ-*.js` and `internal/ui/components/scripts_bundle.go`.
5. In your own templates, JavaScript and CSS, replace `data-tui-*` with `data-slot` (for example `[data-slot=dialog-content]`), open state with `data-open` and `data-closed`, and `window.tui` with `window.templ`. Nothing fails to compile here, so search for `data-tui` and `window.tui`.
6. `API_URL` is new and optional; it defaults to `APP_URL`.
7. Run `go tool templ generate` and `go mod tidy`, then build and test.

Adding the JSON API, MCP or SvelteKit to an existing project is a new generation with the same answers plus the new flags; `goilerplate update` keeps your current selection.

## v3.0.3

September 7, 2026

**In short:** The Admin link is back in Paid projects without Workspaces.

### Fixed
- **Admin link**: Paid projects without Workspaces show the Admin menu link again.

### Upgrade
Run `goilerplate update`.

## v3.0.2

September 4, 2026

**In short:** Free projects can pick Datastar, as promised.

### Fixed
- **`goilerplate new --framework datastar`** no longer rejects Free with `Free uses SQLite, SMTP, htmx, and no paid modules`. Free has included Datastar 1.0 since September 1; the CLI still carried the August rule.
- **The wizard** asks for the frontend for Free too, instead of always generating htmx.
- Reported and fixed by Tom Siouan in the first community pull request. Thank you.

### Upgrade
`go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest`.

## v3.0.1

September 4, 2026

**In short:** `goilerplate doctor` checks the `.env` you actually have.

### Fixed
- **`goilerplate doctor`** no longer fails every fresh v3 project with `missing required values: APP_NAME`. Since v3.0.0 the project name lives in `internal/brand`; doctor now requires exactly the keys your stack ships in `.env.example`.
- Reported by an Early Access developer within hours of launch. Thank you.

### Upgrade
`go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest`. If you added `APP_NAME` to `.env` to get past the check, remove it; nothing reads it.

## v3.0.0

September 3, 2026

**In short:** goilerplate v3 is a generator. Pick the stack, run one command, get a clean Go repository with the hard parts already connected and none of the code you left out.

### New
- **The generator**: start in the web configurator or run `goilerplate new`. Choose htmx 4 or Datastar 1, SQLite or PostgreSQL, Stripe or Polar, SMTP or Resend, and add Workspaces, OAuth, S3-compatible storage, a Markdown blog and Markdown docs only when you need them.
- **Only what you chose**: unselected modules, dependencies, environment variables and tests stay out, and every project starts with one tailored `001_initial_schema.sql`.
- **Production foundations, already wired**: passwords, magic links, verification, recovery, OAuth, revocable sessions, workspaces, roles, invitations, tenant isolation, billing, audit history, storage, content, SEO, legal pages, admin, analytics hooks, structured logs, Sentry, health checks, graceful shutdown, CSRF, security headers, rate limits, validation, selection-aware tests and GitHub Actions CI. shadcn-templ components are vendored: open the file and change it.
- **Updates that stay in Git**: `goilerplate update` regenerates your exact configuration on its own branch, so every change is visible before it reaches your code. `goilerplate.lock` records the template version and the stack.
- **Ready for coding agents**: `AGENTS.md`, `llms.txt` and Markdown docs from the first prompt.
- **Free is real**: $0 forever with unlimited local projects: Go, templ, Tailwind CSS, shadcn-templ, htmx 4 or Datastar 1, SQLite, SMTP, authentication, security defaults, tests and CI.

### Changed
- **Earlier customers are already in**: every goilerplate v1, v2 and templUI Pro license is a Paid v3 license with lifetime updates. No new purchase, no renewal.

### Upgrade
Install the CLI: `go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest`, then `goilerplate new`. The legacy repository stays archived and read-only with its releases and tags.

## v2.3.0

August 15, 2026

**In short:** goilerplate focuses on the Go SaaS backend stack.

### Changed
- **The stack lives at the repository root**, not in `boilerplate/`. This breaks your next `git pull upstream main`; see the release for the steps.
- **templ v0.3.1020**, so fresh setups no longer crash on the first `task dev`.

### Removed
- **Premium blocks and the website login.** Nothing is revoked: `v2.2.0` and every earlier release still include all 222 blocks, in the history and your fork. Open-source blocks are on [shadcn-templ.com](https://shadcn-templ.com/?ref=goilerplate-changelog).

### Upgrade
Pull the release into your fork: `git pull upstream main`, then move your code from `boilerplate/` to the root.

## v2.2.0

June 1, 2026

**In short:** Logs you can actually read: pretty and colored in development, structured JSON in production, set from `.env`.

### New
- **Colored console logs** in development; color drops off when output is not a terminal.
- **`LOG_LEVEL`**, separate from `APP_ENV`: more detail without development mode. Empty means debug in development and info in production.
- **`LOG_FORMAT`**, `pretty` or `json`, for tools that scrape stdout. Defaults to pretty.
- **`LOG_HTTP`** turns request logs off, for when your proxy already logs them.

### Changed
- **Sentry** is a separate handler, so pretty console logs and Sentry run side by side.
- **Static assets are never logged**: `/assets`, `/templui`, `/uploads` and the favicon.
- **templUI v1.12.0.**

### Upgrade
`git pull upstream main`.

## v2.1.0

May 29, 2026

**In short:** AI-ready out of the box: open the boilerplate in your coding tool and it knows its way around from minute one.

### New
- **`boilerplate/AGENTS.md`**: a lean project map for coding tools: the stack, the layers, the path for a new feature, goose migrations, configuration and the important code locations. It links to source instead of duplicating it.
- **An `AGENTS.md` for the block library**, so agents know the 222+ blocks too.
- **The developer docs ship inside the boilerplate**, for you and your agent, offline.
- **Custom rules**: `AGENTS.md` starts with a block for your conventions, which updates keep through a normal Git merge. Works with any LLM.

### Upgrade
`git pull upstream main`.

## v2.0.0

May 22, 2026

**In short:** templUI Pro and goilerplate are one product now, called goilerplate.

### New
- **Everything in one**: the Go and templ boilerplate with auth, billing and dashboards, plus 222+ premium blocks in 33 categories. One license, one repository, lifetime updates.

### Upgrade
Nothing to do. License terms are unchanged.

## v1.9.0

May 4, 2026

**In short:** Stripe Managed Payments: Stripe is the merchant of record and handles VAT, GST and US sales tax.

### New
- **Stripe Managed Payments** is on when you select Stripe, in the countries Stripe supports, for digital products: SaaS, software, courses, e-books and electronically delivered services. No extra environment flag.

### Changed
- **stripe-go v85**, up from v81. Polar stays the default provider.

### Upgrade
`git pull upstream main`, then accept the Managed Payments terms in your Stripe Dashboard and give each Stripe product a digital-product tax code.

## v1.8.0

May 4, 2026

**In short:** One S3-compatible storage for development and production, and no Docker for the default setup.

### Changed
- **One storage provider** in development and production, for real parity. Cloudflare R2 is recommended; any S3-compatible service works.
- **A simpler storage block** in `.env.example`, a shorter dev task flow and a short cloud quickstart.

### Removed
- **Bundled MinIO and the dev `docker-compose.yml`**: SQLite plus cloud S3 needs no Docker.

### Upgrade
`git pull upstream main`, then set your S3 storage in `.env`.

## v1.7.3

April 15, 2026

**In short:** CSS stays cached between deploys.

### Fixed
- **Cache-busting for CSS**: Cloudflare and browsers cache static files properly between deploys.

### Upgrade
`git pull upstream main`.

## v1.7.2

April 1, 2026

**In short:** templUI v1.9.3 and the goals dialogs work again.

### Changed
- **templUI v1.9.3.**

### Fixed
- **Goals flow dialogs** work with the latest templUI dialog system.

### Upgrade
`git pull upstream main`.

## v1.7.1

March 18, 2026

**In short:** A table of contents on docs pages.

### New
- **Table of contents** on the right of docs pages, for faster scanning and one-click in-page navigation.

### Changed
- **templUI v1.9.0.**

### Upgrade
`git pull upstream main`.

## v1.7.0

March 15, 2026

**In short:** The templUI v1.8 import workflow, without vendored templUI files.

### Changed
- **templUI v1.8 imports**: shared component scripts and assets work without vendored local templUI files.
- **The Docker and dev build** let Tailwind pick up imported templUI components out of the box.

### Upgrade
`git pull upstream main`.

## v1.6.9

February 16, 2026

**In short:** A public changelog page in the docs.

### New
- **Changelog page** in the docs, on its own route.

### Changed
- **templUI v1.5.0**, for more consistent interactive components.

### Fixed
- **Favicon and avatar scripts** load reliably.

### Upgrade
`git pull upstream main`.

## v1.6.8

February 10, 2026

**In short:** CSS and JS changes show up right away in development.

### Changed
- **A faster frontend loop**: CSS and JS changes show immediately, with no restart.

### Upgrade
`git pull upstream main`.

## v1.6.7

February 5, 2026

**In short:** templUI v1.4.0 and safer links.

### Changed
- **templUI v1.4.0**, for better UI components.
- **Safer default link behavior.**

### Upgrade
`git pull upstream main`.

## v1.6.6

January 12, 2026

**In short:** Umami Cloud no longer breaks in production.

### Fixed
- **CSP for Umami Cloud**, so analytics keep working in production.

### Upgrade
`git pull upstream main`.

## v1.6.5

January 11, 2026

**In short:** Clearer license wording for teams and client work.

### Changed
- **Commercial license wording** for teams and client work.

### Upgrade
Nothing to do.

## v1.6.4

January 6, 2026

**In short:** templUI v1.1.0.

### Changed
- **templUI v1.1.0.**

### Upgrade
`git pull upstream main`.

## v1.6.3

December 24, 2025

**In short:** Smoother templUI tooling.

### Changed
- **templUI tooling**, for smoother project generation and upgrades.

### Upgrade
`git pull upstream main`.

## v1.6.2

December 10, 2025

**In short:** Starts reliably when storage is slow.

### Fixed
- **Startup** waits when S3 or MinIO is slow to become available, in development and production.

### Upgrade
`git pull upstream main`.

## v1.6.1

December 10, 2025

**In short:** The right client IP behind proxies.

### Fixed
- **Client IPs behind Cloudflare, nginx and other proxies**, for accurate logs and security checks.

### Upgrade
`git pull upstream main`.

## v1.6.0

December 9, 2025

**In short:** The S3 bucket is created on startup.

### New
- **Automatic S3 bucket creation** on startup.

### Fixed
- **CSP for Umami Cloud.**

### Upgrade
`git pull upstream main`.

## v1.5.1

December 8, 2025

**In short:** Landing page section links work from every page.

### Fixed
- **Anchor links** to landing page sections work from all pages.

### Upgrade
`git pull upstream main`.

## v1.5.0

December 4, 2025

**In short:** Umami is the default analytics, privacy first.

### New
- **Umami** as the default analytics provider, with privacy-first defaults.

### Upgrade
`git pull upstream main`.

## v1.4.0

December 1, 2025

**In short:** Better GDPR guidance for analytics.

### Changed
- **GDPR guidance and defaults** for analytics.

### Fixed
- **Google Analytics CSP** for regional domains.

### Upgrade
`git pull upstream main`.

## v1.3.0

November 18, 2025

**In short:** A faster dev workflow and easier first commands.

### Changed
- **Faster local iteration** and easier onboarding commands for new projects.

### Upgrade
`git pull upstream main`.

## v1.2.3

November 14, 2025

**In short:** Clearer setup docs and a fixed typo.

### Changed
- **License and README wording**, for faster setup.

### Fixed
- **A payment provider typo** in `.env.example`.

### Upgrade
`git pull upstream main`.

## v1.2.2

November 10, 2025

**In short:** Go 1.25 and a simpler first run.

### Changed
- **Go 1.25.**
- **A simpler first-run setup.**

### Upgrade
Install Go 1.25, then `git pull upstream main`.

## v1.2.1

November 10, 2025

**In short:** A maintenance release, nothing changes for you.

### Upgrade
Nothing to do.

## v1.2.0

November 10, 2025

**In short:** A more consistent local setup with modern Go tooling.

### Changed
- **Modern Go tooling conventions** for the local setup.

### Upgrade
`git pull upstream main`.

## v1.1.1

November 9, 2025

**In short:** Cleaner auth and settings forms.

### Fixed
- **Auth and settings forms**: cleaner dialogs and form behavior.

### Upgrade
`git pull upstream main`.

## v1.1.0

November 9, 2025

**In short:** Stripe as a second billing provider.

### New
- **Stripe** as an alternative to Polar, which stays the default. One payment interface makes switching easy.

### Upgrade
`git pull upstream main`.

## v1.0.5

November 8, 2025

**In short:** Fresher assets after every deploy.

### New
- **Cache busting** for frontend assets.

### Changed
- **Branding and mobile navigation polish.**

### Upgrade
`git pull upstream main`.

## v1.0.4

November 10, 2025

**In short:** A UI polish release.

### Changed
- **Fresher styles** and trust-oriented branding.

### Upgrade
`git pull upstream main`.

## v1.0.3

November 10, 2025

**In short:** Stronger auth.

### Changed
- **Auth hardening**: stricter validation and stronger password rules.
- **Inline auth feedback** across the core flows.

### Upgrade
`git pull upstream main`.

## v1.0.2

November 10, 2025

**In short:** Docs and footer links work.

### Fixed
- **Documentation and footer links.**

### Upgrade
`git pull upstream main`.

## v1.0.1

November 10, 2025

**In short:** A better quick start.

### Changed
- **Quick-start guidance** and a first frontend performance pass.

### Upgrade
`git pull upstream main`.

## v1.0

October 29, 2025

**In short:** goilerplate launches: a production-ready Go SaaS foundation.

### New
- **Auth**: magic links, passwords and OAuth.
- **Billing** with Polar.
- **Docs, blog, email, file storage and legal pages.**
- **Security defaults**, Docker and PostgreSQL.

### Upgrade
Clone the repository and follow the README.

<details>
<summary>templUI Pro archive</summary>

The following entries are the historical changelog for templUI Pro, which was consolidated into goilerplate in v2.0.0 above. Kept here for reference.

### v1.4.8

April 18, 2026
- Updated to templUI v1.10.0 with the new Range Slider component.
- Replaced dual-slider price range hacks in shop and sidebar filter blocks with the proper range slider.
- Fixed missing slider script on shop, sidebar, and AI blocks so sliders are now interactive.

### v1.4.7

April 7, 2026
- Updated the full block library for compatibility with the latest templUI release.
- Migrated the repo from copied CLI components to templUI's import-based workflow for simpler setup and maintenance.
- Adapted icons and interactive component wiring to the current templUI APIs, including the new class-based icon pattern.

### v1.4.6

February 8, 2026
- Improved consistency for block/category pages across overview, detail, and SEO surfaces.
- Better AI-assisted customization workflow with cleaner AGENTS guidance.
- Fixed countdown preview alignment and avatar script behavior.

### v1.4.5

January 12, 2026
- Clearer commercial licensing terms for teams and agencies.
- Updated core components to templUI v1.1.0.

### v1.4.4

November 20, 2025
- Maintenance release to align package internals and delivery consistency.
- No action required for existing paid users.

### v1.4.3

November 18, 2025
- Better Team/Enterprise setup guidance and fork workflow docs.
- Clearer license wording for commercial usage.

### v1.4.2

November 2, 2025
- Compatibility patch so all blocks/pages keep working on latest templUI updates.

### v1.4.1

November 2, 2025
- Updated to templUI v0.99.0 with broad component quality improvements.

### v1.4.0

October 2, 2025
- Streamlined repo/customer structure for easier access and support.
- Removed legacy Mailchimp and payment paths. The flow stays focused on Stripe.
- Improved release delivery consistency.

### v1.3.1

September 14, 2025
- Fixed invalid nested `<main>` structure in sidebar blocks.

### v1.3.0

September 4, 2025
- Added new Timeline, Social, and Comparison block sets.
- Expanded library to 220+ premium blocks.

### v1.2.1

September 3, 2025
- Migrated key patterns from Modal to Dialog for better accessibility.
- Added 3 advanced sidebar layouts.
- Updated to templUI v0.96.0 with bug and quality improvements.

### v1.2.0

September 3, 2025
- Replaced core modal flows with Dialog patterns for better accessibility and interaction quality.
- Added 3 advanced sidebar layouts for richer team and project navigation.
- Fixed sidebar checkbox and notification-label issues and corrected the default showcase color.

### v1.1.0

August 28, 2025
- Added the new Sidebar system (default/inset/floating + icons-only + tooltips).
- Updated to templUI v0.94.0 with 30+ component improvements.

### v1.0.0

August 6, 2025
- Initial stable launch of templUI Pro.
- Shipped with 210+ premium blocks across 30 categories.
- Introduced Premium Unlimited plan and 50% affiliate program.

</details>
