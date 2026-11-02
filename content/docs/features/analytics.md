---
title: "Analytics"
description: "Optional analytics. Umami. Plausible. Google Analytics."
order: 7
---

## TLDR

Analytics is disabled until you set one of these values:

```env
UMAMI_WEBSITE_ID=
PLAUSIBLE_DOMAIN=
GOOGLE_ANALYTICS_ID=
```

You may enable one provider or several. Most products should start with one.

Analytics belongs to the htmx and Datastar web pages. SvelteKit and Headless projects have no analytics settings. `ROBOTS_NOINDEX=true` still sends `X-Robots-Tag: noindex, nofollow` there.

## Plausible

```env
PLAUSIBLE_DOMAIN=app.example.com
PLAUSIBLE_HOST=plausible.io
```

For self-hosted Plausible, set your own host without `https://`.

## Umami

```env
UMAMI_WEBSITE_ID=your-website-id
UMAMI_HOST=cloud.umami.is
```

Set `UMAMI_HOST` to the hostname of your self-hosted instance when needed.

## Google Analytics

```env
GOOGLE_ANALYTICS_ID=G-XXXXXXXXXX
```

## Security policy

The base layout renders scripts only for configured providers. Security headers add the configured hosts to the Content Security Policy.

Do not paste extra inline scripts into pages. Add a typed config field. Render it through the shared analytics layout. Update the CSP test.

## Privacy

A cookie banner may be required. It depends on the provider and configuration. It also depends on the jurisdiction and data you collect. The included legal pages are placeholders. They are not legal advice.
