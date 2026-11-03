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
- Then check what you changed, and only that: a small change gets a quick look at the changed page or generated project; a release with many changes gets the full run (generate every stack, click through it in Chromium and WebKit).

## Git

- Use a feature branch and pull request for substantial work.
- Keep commits attributed only to the human author. Do not add automation attribution.
- Commits follow Conventional Commits: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, with an optional scope like `feat(cli):`. A short first line, then a sentence or two of why when it helps.
- Commits and pull requests carry no AI attribution: no `Co-Authored-By` for a tool, no "Generated with" line.
- Release notes follow one format, the same in goilerplate, ops and shadcn-templ, so the eye finds things fast. The same text is the GitHub release (which `goilerplate changelog` shows) and the newest section of `CHANGELOG.md` (`## v…`, the date like `November 3, 2026`, then the notes), which `/changelog` shows:
  - First line: `**In short:**` and one sentence, what is new and why to update.
  - Then these sections in this order, empty ones left out: `### New` (what you can do now), `### Changed` (what behaves differently, everything that could surprise), `### Fixed` (what was broken and works now), `### Removed` (what is gone, and what to use instead), `### Upgrade` (always there: how to update, the exact command).
  - One bullet per thing: `- **Name**: one sentence of what the user gets.` Commands in backticks. What the user sees, never internals or commit hashes.
  - A big turning point may add a short `### Why` under the first line. Longer stories go in a blog post or the docs, linked.

## Prose

- Write project text naturally in the project owner's voice.
- Do not use em dashes or standalone hyphens as sentence separators.
