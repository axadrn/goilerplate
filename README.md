# Goilerplate

The public CLI for Goilerplate.

Development is in progress. The first package provides GitHub Device Flow login,
local session storage, `whoami`, and logout. The CLI requests only GitHub's
`user:email` scope. It never asks for repository access and never stores the
temporary GitHub OAuth token.

```text
goilerplate login
goilerplate whoami
goilerplate logout
```

The generator, update command, and release installation flow will land in the
next packages.
