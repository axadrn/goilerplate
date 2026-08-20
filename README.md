# Goilerplate

The public CLI for Goilerplate.

Development is in progress. The first package provides GitHub Device Flow login,
local session storage, authenticated project generation, `whoami`, and logout. The CLI requests only GitHub's
`user:email` scope. It never asks for repository access and never stores the
temporary GitHub OAuth token.

```text
goilerplate login
goilerplate whoami
goilerplate new --module example.com/acme ./acme
goilerplate logout
```

Paid selections stay explicit and composable:

```text
goilerplate new --edition paid --module example.com/acme --database postgres --teams --oauth google,github ./acme
```

The update command and release installation flow will land in the next packages.
