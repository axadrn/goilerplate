---
title: Privacy Policy
description: What goilerplate collects. Almost nothing.
lastUpdated: November 1, 2026
---

This policy covers goilerplate.com and the goilerplate CLI. It does not cover applications that you generate and operate yourself.

## Controller

Axel Adrian is responsible for this site. Privacy questions can be sent to [{{support_email}}](mailto:{{support_email}}).

## The website

goilerplate.com is a static site. There are no accounts, no forms, no analytics and no advertising or tracking cookies.

The site runs on its own server behind Cloudflare. Like any web server, both process your network address and basic request data such as the path, time and browser, to deliver pages and stop abuse. These logs are kept only as long as needed for security and troubleshooting.

The site stores only your light or dark theme choice in your browser's local storage. It stays in the browser and is never sent anywhere. You can remove it through the browser's site-data controls.

Following a link to GitHub or another site takes you to that provider. Their own privacy terms apply there.

## The CLI

The goilerplate CLI has no telemetry and no account. `goilerplate new` runs entirely on your machine.

Two commands use the network:

- `goilerplate changelog` reads the public release notes from the GitHub API.
- `goilerplate update` downloads the goilerplate release your project was made with from the Go module proxy, `proxy.golang.org` unless you set `GOPROXY`.

Those requests go straight to GitHub and to the proxy you use. goilerplate never sees them. Your project and its source code never leave your machine.

## Services involved

- [Cloudflare](https://www.cloudflare.com/privacypolicy/) delivers and protects goilerplate.com.
- [GitHub](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement) hosts the code, issues and release notes.
- The [Go module proxy](https://proxy.golang.org/privacy) serves goilerplate releases for `go install` and `goilerplate update`.

Some providers may process data outside the European Economic Area under their published transfer safeguards. Their linked policies contain the current details.

## Your rights

Depending on applicable law, you may request access, correction, deletion, restriction or portability, or object to processing. Email [{{support_email}}](mailto:{{support_email}}).

You may also lodge a complaint with the data protection authority responsible for your place of residence or work.
