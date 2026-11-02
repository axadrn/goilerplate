---
title: "Philosophy"
description: "Clear code. Boring tools. Ownership without lock-in."
order: 6
---

## The rule

Clear beats clever.

goilerplate chooses ordinary Go and explicit code because the person maintaining the product six months later matters more than the person admiring an abstraction today.

## Boilerplate, not a framework

The generator writes a starting repository. The generated application does not import a proprietary goilerplate runtime.

You can change or remove anything. The cost is visible because the code is visible.

## Generate less

Unselected features should not exist in the project.

Choose SQLite. The PostgreSQL driver and setup stay out. Leave Workspaces disabled and its code stays out. This is simpler than runtime feature switches.

## One real path

The interactive CLI and explicit flags produce the same generator config. CI and the website builder use it too. There is one validation path. One generation path.

## Small boundaries

Interfaces exist where a provider or frontend is genuinely interchangeable. Payment is one boundary. Mail is another. So are storage and frontend responses.

There is no generic abstraction for models, repositories, handlers, or permissions.

## Secure defaults

Security should not depend on every future developer remembering every checklist. Revocable sessions. CSRF protection. Secure cookies. Input limits. Rate limits. Trusted proxy configuration. Conservative browser headers. The foundation includes them all.

Defaults reduce risk. They do not replace a security review for your product.

## Local ownership

Generation and updates never require repository access. Your source code stays on your machine.

`goilerplate update` downloads the template of your old version, renders the old and new trees on your machine, and lets Git merge them. Conflicts remain normal Git conflicts.

## Boring operations

The default deployment is one Go binary in one container. SQLite is a valid production choice for a single writable application instance. PostgreSQL is available when the product or operating model needs it.

Start with the smallest system that safely serves the real workload. Add infrastructure after evidence.
