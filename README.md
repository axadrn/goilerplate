# goilerplate

**The Go SaaS starter kit, free and open source.** One command generates a production Go app with login, billing, teams, admin, an API and tests, right on your machine. Every line is yours.

[Website](https://goilerplate.com) · [Docs](https://goilerplate.com/docs) · [Changelog](https://goilerplate.com/changelog)

## What you get

- **Auth:** passwords, magic links, Google and GitHub OAuth, revocable sessions
- **Billing:** Stripe or Polar, subscriptions, customer portal, signed webhooks
- **Teams:** workspaces, roles, invitations, seat limits
- **Admin:** user search, metrics, session revocation, support impersonation
- **API:** JSON API, personal API tokens, MCP server for your customers' agents
- **Production:** tests, CI, Dockerfile, migrations, Sentry, a stack-aware `AGENTS.md`

Stack: Go, templ, Tailwind CSS, shadcn-templ, with htmx, Datastar, SvelteKit or headless. SQLite or PostgreSQL.

## Quick start

You need Go 1.26+ and Git.

```bash
go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest
goilerplate new
```

`goilerplate new` without options opens an interactive setup. It renders the project on your machine: no account, no network.

## Commands

| Command | What it does |
|---|---|
| `new` | Generate a new project |
| `update` | Update a generated project on a new Git branch, from v4 on |
| `doctor` | Check a project's tools and `.env` |
| `changelog` | Show release notes |
| `version` | Show the CLI version |

## Options for `new`

```bash
goilerplate new --module example.com/acme --framework htmx --database sqlite --payment stripe --workspaces ./acme
```

| Option | Values | Default |
|---|---|---|
| `--framework` | `htmx`, `datastar`, `svelte`, `headless` | `htmx` |
| `--database` | `sqlite`, `postgres` | `sqlite` |
| `--payment` | `stripe`, `polar` | `stripe` |
| `--mail` | `smtp`, `resend` | `smtp` |
| `--oauth` | `google`, `github` (comma separated) | none |
| `--content` | `blog`, `docs` (comma separated) | none |
| `--workspaces` | include teams | off |
| `--storage` | include S3 file storage | off |
| `--api` | add the JSON API | off |
| `--mcp` | add the MCP server, needs `--api` | off |
| `--headless` | Go backend with the JSON API, no frontend | off |

SvelteKit projects also need Node.js 22.17+ and pnpm.

## License

- The CLI and this repository are [MIT](LICENSE).
- The template in `generator/_template/` and every project goilerplate generates are [0BSD](LICENSE-0BSD). Use, change and sell what you generate. No notice or attribution is required, and generated projects ship without a LICENSE file, so you pick your own.
- Contributions come in under the license of the files they change. No CLA.
- The licenses cover the code, not the name goilerplate or its logo.
