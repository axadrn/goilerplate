# AGENTS.md

## Scope

This repository contains the public Goilerplate CLI and the shared API contract. It never contains the private project template.

## Engineering

- Prefer the Go standard library.
- Keep packages small and concrete. Add interfaces only at real I/O boundaries used by tests.
- Keep request and response types in `api`. The private service imports this package, never the reverse.
- Never log or persist GitHub OAuth tokens. Store only Goilerplate session tokens, with file mode `0600` inside a `0700` configuration directory where Unix permissions apply.
- Run `go test -race ./...`, `go vet ./...`, `go build ./...`, and `go mod tidy -diff` before pushing.

## Git

- Use a feature branch and pull request for substantial work.
- Keep commits attributed only to the human author. Do not add automation attribution.

## Prose

- Write project text naturally in the project owner's voice.
- Do not use em dashes or standalone hyphens as sentence separators.
