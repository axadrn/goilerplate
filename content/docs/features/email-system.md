---
title: "Email System"
description: "Transactional email through SMTP or Resend. Optional newsletter storage."
order: 3
---

## TLDR

The SMTP or Resend provider selected during generation delivers email in every environment.

## Development

Configure the generated provider settings in `.env`, then run `task dev`. Missing or invalid email configuration stops the application at startup. Use separate test credentials locally when your provider offers them. This keeps local delivery on the same code path as production and exposes provider, authentication, TLS, and network failures before deployment.

## SMTP

SMTP works with any standard provider:

```env
EMAIL_FROM=noreply@example.com
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_PASSWORD=...
SMTP_TLS=starttls
```

Supported TLS modes are `starttls`, `tls`, and `none`.

Username and password must either both be present or both be empty.

## Resend

```env
EMAIL_FROM=noreply@example.com
RESEND_API_KEY=re_xxx
RESEND_AUDIENCE_ID=
```

Email delivery requires the API key. The Audience ID is optional and used only for newsletter contacts.

## Included transactional flows

The template sends exactly these emails:

- Sign-in link. Opening it also verifies the address.
- Welcome, after onboarding sets the name.
- Password reset. A one-time sign-in link that removes the old password.
- Email change verification, sent to the new address.
- Email change notice, sent to the old address.
- Account deleted.
- Workspace invitation, when Workspaces is selected.

Each message receives plain text and HTML content through the same service boundary. Action emails include a primary button and a visible fallback URL.

## Newsletter boundary

Transactional mail and newsletter subscription are separate packages.

With SMTP, newsletter signups are stored idempotently in the local `newsletter_subscribers` table.

With Resend and an Audience ID, the signup is sent to Resend Contacts. The application does not implement a campaign sender.

## Error behavior

Provider errors return to the calling service. Account flows do not claim success when the required email could not be sent.

Logs record the provider and operation without recipient, message body, or action token attributes.

## Customize

Change message copy in `internal/service/email_templates.go` and the shared layout in `internal/service/email_renderer.go`. Change provider behavior in the selected mailer package. Authentication code should not import an SMTP or Resend client directly.
