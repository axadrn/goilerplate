# goilerplate

The public CLI for goilerplate.

Development is in progress. The CLI signs in through GitHub, generates projects,
and manages account access. It requests only GitHub's `user:email` scope. It never
asks for repository access and never stores the temporary GitHub OAuth token.

```text
goilerplate login
goilerplate whoami
goilerplate activation resend
goilerplate new
goilerplate update
goilerplate logout
```

## Create a project

Run one command:

```text
goilerplate new
```

The interactive setup asks where the project should live, what its Go module
path is, and which features you want. Use the arrow keys to move, Space to
toggle multiple choices, and Enter to continue. Nothing is downloaded until
you review the choices and select Generate.

The setup is intentionally thin. It builds the same arguments as the regular
command and hands them to the same validation and generation code. There is one
generation path, whether you use the interactive setup, copy a command from the
website, or run goilerplate in CI.

Repeatedly pressing Enter creates the standard Free project in `./my-app` with
module path `example.com/my-app`. Change those two values for a real project.

Prefer flags for scripts and repeatable builds:

```text
goilerplate new --name Acme --module example.com/acme ./acme
goilerplate new --edition paid --module example.com/acme --database postgres --teams --oauth google,github ./acme
```

When input or output is redirected, `goilerplate new` requires these flags and
never tries to open an interactive screen.

Free always uses SQLite, SMTP, and the htmx foundation. Paid shows every
optional module. The interactive setup does not contain the private template
and your source code never leaves your machine.

## One company, one license, one login per developer

Think of a license as one shared key ring for a company:

1. The company owns the license.
2. Every developer signs in with their own GitHub account.
3. Nobody shares a password, personal token, or login session.
4. Owners invite and remove developers.
5. Owners create CI keys for machines. A CI key can generate code, but it cannot manage people or other keys.

`goilerplate whoami` prints your license ID. Use that ID in the management commands:

```text
goilerplate license members <license-id>
goilerplate license invite <license-id> developer@example.com
goilerplate license invite --owner <license-id> cofounder@example.com
goilerplate license remove <license-id> <user-or-invitation-id>

goilerplate token create <license-id> deploy
goilerplate token list <license-id>
goilerplate token revoke <license-id> <token-id>
```

A new CI key is shown once. Store it in the CI provider's secret store. If a developer leaves, remove the member. goilerplate revokes the company's CI keys so the Owner can create clean replacements.

In CI, expose the key only to the command that needs it:

```text
GOILERPLATE_TOKEN="$GOILERPLATE_TOKEN" goilerplate new --module example.com/acme ./acme
```

`GOILERPLATE_TOKEN` works only for generation and later update commands. Account and license management still require a personal Owner login.

Account deletion requires your GitHub login as confirmation. Capitalization and a leading `@` do not matter:

```text
goilerplate account delete --confirm axadrn
```

A company license always keeps at least one Owner. The final Owner must add another Owner before deleting their account.

### Why can one account show more than one license?

Access belongs to the license, not to the login. A developer can work for two companies, so one personal login can access both company licenses. A personal Free license can also remain beside a Paid company license. goilerplate marks the best active license with `*` and uses it automatically. If the developer leaves the company later, their personal Free access still works.

This does not create duplicate billing. Each Paid license still belongs to exactly one company.

Paid selections stay explicit and composable:

```text
goilerplate new --edition paid --module example.com/acme --database postgres --teams --oauth google,github ./acme
```

## Update an existing project

Run this inside a generated project:

```text
goilerplate update
```

The short version:

1. Your project contains `goilerplate.lock`. It says which template version and answers created the project.
2. goilerplate downloads that old generated tree and the newest generated tree.
3. Git compares the old tree, your current project, and the new tree.
4. goilerplate creates a new branch such as `goilerplate-update-v3.1.0`.
5. Your current branch and working files stay untouched.

If Git finds no conflicts, switch to the new branch and review it. If Git finds conflicts, the new branch contains normal conflict markers. Resolve them exactly like any other Git merge.

```text
git switch goilerplate-update-v3.1.0
```

To abandon the update, stay on your original branch and delete the update branch.

```text
git branch -D goilerplate-update-v3.1.0
```

`goilerplate update` requires a clean Git worktree and Paid license access. Your source code never leaves your machine. The service receives only the answers already stored in `goilerplate.lock` and returns generated template trees.

The release installation flow will land in a later package.
