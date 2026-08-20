# Goilerplate

The public CLI for Goilerplate.

Development is in progress. The CLI signs in through GitHub, generates projects,
and manages account access. It requests only GitHub's `user:email` scope. It never
asks for repository access and never stores the temporary GitHub OAuth token.

```text
goilerplate login
goilerplate whoami
goilerplate activation resend
goilerplate new --module example.com/acme ./acme
goilerplate logout
```

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

A new CI key is shown once. Store it in the CI provider's secret store. If a developer leaves, remove the member. Goilerplate revokes the company's CI keys so the Owner can create clean replacements.

In CI, expose the key only to the command that needs it:

```text
GOILERPLATE_TOKEN="$GOILERPLATE_TOKEN" goilerplate new --module example.com/acme ./acme
```

`GOILERPLATE_TOKEN` works only for generation and later update commands. Account and license management still require a personal Owner login.

Account deletion requires the exact GitHub login as confirmation:

```text
goilerplate account delete --confirm axadrn
```

A company license always keeps at least one Owner. The final Owner must add another Owner before deleting their account.

### Why can one account show more than one license?

Access belongs to the license, not to the login. A developer can work for two companies, so one personal login can access both company licenses. A personal Free license can also remain beside a Paid company license. Goilerplate marks the best active license with `*` and uses it automatically. If the developer leaves the company later, their personal Free access still works.

This does not create duplicate billing. Each Paid license still belongs to exactly one company.

Paid selections stay explicit and composable:

```text
goilerplate new --edition paid --module example.com/acme --database postgres --teams --oauth google,github ./acme
```

The update command, interactive interface, and release installation flow will land in later packages.
