---
title: "SEO, Metadata, and Sitemap"
description: "Canonical URLs. Social previews. Robots rules. Sitemap generation. Structured content."
order: 6
---

## TLDR

Every public page passes one small SEO value to the base layout:

```go
@layouts.Base(layout.SEOProps{
    Title:       "Pricing",
    Description: "Simple pricing for Acme.",
    Path:        "/pricing",
})
```

The layout renders the title and description. The canonical URL. Open Graph tags. The Twitter card.

Site-wide identity lives in `internal/brand/brand.go`. Change the product name, tagline, and default description there. Do not add brand Env keys.

## Required fields

- `Title`: specific page title.
- `Description`: plain summary for search and sharing.
- `Path`: canonical path beginning with `/`.

Optional fields choose a custom social image and Open Graph type.

The layout combines page metadata with the owned brand values. Page copy stays next to the page. Shared identity stays in `internal/brand`.

## Canonical origin

`APP_URL` is the canonical origin:

```env
APP_URL=https://example.com
```

Set it correctly in production. A wrong value produces wrong canonical and social URLs.

## Sitemap

The sitemap service combines known static routes with selected Markdown blog and documentation pages.

```text
GET /sitemap.xml
```

Deleted or private routes should not enter the sitemap.

## Robots

`robots.txt` is generated at runtime so its sitemap URL uses the current public origin.

```text
GET /robots.txt
```

For private beta deployments, add `X-Robots-Tag: noindex` at the reverse proxy too. A robots file is a request. It is not access control.

## Markdown content

Frontmatter supplies titles and descriptions for blog and docs pages. Slugs come from repository paths. This keeps URLs stable and reviewable.

## Social images

The default image is embedded at `assets/img/og.png`. Replace it with your own brand image before launch. Keep the common 1200 by 630 pixel format. Add page-specific images only when they materially improve sharing.

The browser icon lives at `assets/img/favicon.svg`. Replace it with your own SVG while keeping the filename unchanged.

## Checklist

1. Unique title and description.
2. Correct canonical path.
3. One clear page heading.
4. Useful internal links.
5. Sitemap contains the route.
6. Private pages are not indexed.
7. Redirect old public slugs instead of returning 404.
