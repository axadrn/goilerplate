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
- Commits follow Conventional Commits: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, with an optional scope like `feat(cli):`. A short first line, then a sentence or two of why when it helps.
- Commits and pull requests carry no AI attribution: no `Co-Authored-By` for a tool, no "Generated with" line.
- Release notes have three sections, empty ones left out: `### New`, `### Fixed`, `### Upgrade` (how to update, one line). They come from the `feat:` and `fix:` commits.

## Prose

- Write project text naturally in the project owner's voice.
- Do not use em dashes or standalone hyphens as sentence separators.
