# AGENTS.md

## Scope

This repository is goilerplate, free and open source: the CLI, the generator with its template, and the website. Everything runs on the user's machine. There is no goilerplate server, account or license.

## Brand

- Always write the product name as `goilerplate` in lowercase, including at the start of sentences and headings.
- Keep technical identifiers unchanged.

## Engineering

- Prefer the Go standard library.
- Keep packages small and concrete. Add interfaces only at real I/O boundaries used by tests.
- `generator` owns the template, the configuration and `goilerplate.lock`. The CLI uses its types directly.
- The CLI makes no network call except `changelog` (GitHub releases) and `update`, which downloads a released template from the Go module proxy. No telemetry.
- Nothing private enters this repository: no secrets, no customer data, no server names or internal plans.
- Run `go test -race ./...`, `go vet ./...`, `go build ./...`, and `go mod tidy -diff` before pushing.

## Git

- Use a feature branch and pull request for substantial work.
- Keep commits attributed only to the human author. Do not add automation attribution.

## Prose

- Write project text naturally in the project owner's voice.
- Do not use em dashes or standalone hyphens as sentence separators.
