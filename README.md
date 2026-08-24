# goilerplate

Generate a production-ready Go SaaS, own every line, and update it with Git.

Development is in progress. Public launch is coming soon. Visit [goilerplate.com](https://goilerplate.com), then star or watch this repository to follow the release.

## Install

Homebrew includes the binary, so Go does not need to be installed first:

```text
brew install --cask axadrn/tap/goilerplate
```

If Go is already installed:

```text
go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest
```

Every release includes ready-to-run binaries for macOS and Linux on amd64 and arm64. On Windows, use WSL and install the Linux build inside it. Native Windows and PowerShell are not supported.

## Create a project

```text
goilerplate login
goilerplate new
```

`new` opens a small terminal wizard. Choose a project name, module path, database, frontend, and the features you need. Review the command, press Enter, and get an ordinary Go repository on your machine.

Prefer explicit flags for scripts or CI:

```text
goilerplate new --name Acme --module example.com/acme ./acme
goilerplate new --edition paid --name Acme --module example.com/acme --framework datastar --database postgres --teams ./acme
```

The Free edition is a fixed htmx, SQLite, and SMTP foundation. Paid generation adds htmx or Datastar, PostgreSQL, payment providers, teams, OAuth, storage, content, and the Projects example.

The website builder produces the same flags. The terminal wizard, copied commands, and CI all use one generation path.

## Update a project

Run this inside a generated repository:

```text
goilerplate update
```

goilerplate downloads the old and new generated trees, then asks Git to perform a real three-way merge. The result lands on a new branch such as `goilerplate-update-v3.1.0`. Your current branch and working files stay untouched.

Clean updates are ready to review. Conflicts use normal Git conflict markers. To abort, delete the update branch.

Your source code never leaves your machine. The service receives only the answers already stored in `goilerplate.lock` and returns generated template trees.

## Accounts and company access

Every developer signs in with their own GitHub account. Nobody shares a password or personal token. One company license can cover unlimited internal projects and developers.

```text
goilerplate whoami
goilerplate license members <license-id>
goilerplate license invite <license-id> developer@example.com
goilerplate license remove <license-id> <user-or-invitation-id>
```

`whoami` prints the license ID. Owners invite or remove people. A company always keeps at least one Owner.

CI uses a separate generation-only token:

```text
goilerplate token create <license-id> deploy
GOILERPLATE_TOKEN="$GOILERPLATE_TOKEN" goilerplate new --module example.com/acme ./acme
```

Store the token in the CI provider's secret store. It cannot manage people or other tokens.

## Useful commands

```text
goilerplate login
goilerplate logout
goilerplate whoami
goilerplate claim buyer@example.com
goilerplate new
goilerplate update
goilerplate doctor
goilerplate changelog
goilerplate account delete
```

`doctor` checks the generated repository and local tools without changing anything. `changelog` shows the latest published releases.

## Privacy and scope

The CLI requests GitHub's `user:email` scope for identity. It never asks for repository access, never stores the temporary GitHub OAuth token, and never sends customer source code to goilerplate.

This public repository contains the CLI and shared API contract. The paid application template remains private and is delivered through the generator service.

Read the [documentation](https://goilerplate.com/docs), review the [license](https://goilerplate.com/license), or report a CLI problem in [GitHub Issues](https://github.com/axadrn/goilerplate/issues).

## License

The CLI source is available under the [MIT License](LICENSE). Generated application code follows the product license shown at [goilerplate.com/license](https://goilerplate.com/license).
