---
title: "Markdown Content"
description: "Version-controlled blog and documentation without a CMS"
order: 10
---

## TLDR

Select Blog, Docs, or both. Write Markdown. Commit it. Deploy it.

There is no CMS. No content database. No editor service.

## File layout

```text
content/
  blog/
    first-post.md
  docs/
    introduction.md
    guides/
      _index.md
      deployment.md
```

## Frontmatter

```yaml
---
title: "Deploy the app"
description: "A short production deployment guide"
order: 3
---
```

Blog posts may add metadata supported by the generated model. Date. Author. Tags. Image. Draft state.

## Documentation navigation

Folders form groups. `_index.md` sets the group title. It also sets the description and order. Page `order` values control navigation before the alphabetical fallback.

Previous and next links use the same tree.

## Rendering

Markdown is parsed on the server into sanitized HTML. Code blocks receive syntax highlighting and a copy button. Headings receive stable IDs for links and the page table of contents.

## When this is the right choice

Use repository Markdown when product documentation changes with code and developers own the writing workflow.

Add a CMS when non-developers need drafts, approvals, scheduled publishing, media workflows, or frequent independent changes.

## Remove it

Leave Blog or Docs unselected during generation and that entire surface is omitted. After generation, `goilerplate update` keeps the choices in `goilerplate.lock`. Remove generated modules manually if the product no longer needs them.
