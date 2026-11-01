---
title: "Logging and Observability"
description: "Readable development logs. Structured production logs. Optional Sentry reporting."
order: 8
---

## TLDR

The app uses Go's standard `log/slog` package.

```env
LOG_LEVEL=
LOG_FORMAT=pretty
LOG_HTTP=true
SENTRY_DSN=
```

## Log level

Supported values are `debug`, `info`, `warn`, and `error`.

An empty value defaults to debug in development and info in production.

```env
LOG_LEVEL=debug
```

## Format

Use readable output locally:

```env
LOG_FORMAT=pretty
```

Use structured output for log collectors:

```env
LOG_FORMAT=json
```

## Startup

The process loads and validates application configuration before installing the configured logger. Bootstrap errors use the safe default logger. Startup, runtime, and shutdown after initialization use the configured format and destination.

Configuration validation returns every problem at once. Each problem is logged as its own event, then the process exits with status 1. Configuration code never logs and never exits the process.

## HTTP requests

```env
LOG_HTTP=true
```

Request logs include the method and path, status, duration, and useful server context. They do not log form bodies or cookies. Authorization headers and query secrets stay out too.

## Sentry

Set `SENTRY_DSN` to report server errors. Leave it empty to disable Sentry.

The app still logs locally when Sentry is disabled or unavailable.

## Logging rule

Log what happened. Where it happened. Why it happened. Do not log passwords or tokens. Keep session hashes, OAuth codes, email bodies, signed provider URLs, and full webhook payloads out too.

Email logs contain the selected provider and operation without recipient, message body, or action token attributes.

Use stable field names so production searches remain useful.
