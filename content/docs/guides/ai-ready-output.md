---
title: "AI-ready Output"
description: "Repository instructions for coding agents. Public Markdown documentation for LLM tools."
order: 5
---

## TLDR

Coding agents can ship features in minutes. Do not let them invent your architecture along the way.

Generated projects give coding agents local repository rules and give documentation tools a public, machine-readable map.

These are two separate surfaces:

- `AGENTS.md` explains how to work inside the generated repository.
- `/llms.txt` points external tools to the public product documentation.

## Repository instructions

Every generated repository includes an `AGENTS.md` tailored to the selected stack. It names the architecture, important commands, generated boundaries, and the files that should remain authoritative.

Coding agents read it from the repository. It is not served publicly.

Edit it as the product develops. Add business rules that should survive across tools and sessions.

```md
## Product boundaries

- Keep billing rules in the subscription service.
- Every tenant query requires a workspace ID.
- Run the race-enabled test suite before release.
```

## Public LLM documentation

Projects serve `/llms.txt` as a small index of public product documentation. When the Docs module is selected, every page also has a raw Markdown version.

[Open the live goilerplate llms.txt](/llms.txt)

```text
/llms.txt
/docs.md
/docs/getting-started.md
/docs/features/authentication.md
```

Rendered Docs pages advertise both `/llms.txt` and their Markdown counterpart in the page metadata. Compatible crawlers can discover the machine-readable version without scraping the visual page.

The public index never exposes `AGENTS.md`, source code, environment values, or private operational instructions.

## Make it yours

The files ship as ordinary owned code. Change the copy, add documentation pages, or remove the public route when the product should not publish LLM-readable documentation.

Do not copy goilerplate documentation into the generated project. The included pages are a generic, stack-aware example for documenting the product being built.
