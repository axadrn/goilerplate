---
title: "Workspaces"
description: "Customer workspaces. Roles. Invitations. Tenant isolation. Billing ownership."
order: 3
---

## TLDR

Select Workspaces when customers of your generated SaaS need shared workspaces.

This is not the global operator Admin panel. It is product code for your future users.

## ELI5

Imagine you build project management software.

Acme creates one workspace. Alice and Bob join it. Acme's projects belong to that workspace. So do its members, subscription, and audit history.

Alice can also be invited to another workspace in the same SaaS. The workspace switcher appears only when she has more than one workspace membership.

## Data model

```text
User -> WorkspaceMember -> Workspace -> Subscription
```

Tenant-owned product rows also carry `workspace_id`.

Routes make the active workspace visible:

```text
/app/{workspaceSlug}/projects
/app/{workspaceSlug}/workspace
/app/{workspaceSlug}/billing
```

Middleware resolves the membership before a handler runs. Cross-workspace access returns not found.

## Roles

| Role | Can do |
| --- | --- |
| Owner | Billing, roles, ownership, settings, deletion, and all normal product work |
| Admin | Members, invitations, settings, and normal product work |
| Member | Normal product work |

Multiple Owners are allowed. The final Owner cannot leave until another Owner exists. They also cannot delete their account or be demoted.

Permissions are static capabilities in Go code. There is no database permission builder.

## Onboarding

Normal signup creates one workspace and one Owner membership.

Invitation signup joins the invited workspace and does not create an unwanted personal workspace.

Invitation acceptance checks the current plan and member count in one transaction. Concurrent acceptances cannot bypass the seat limit.

## Billing

With Workspaces selected, the workspace owns the subscription. Only Owners may start checkout or open billing management.

Provider metadata carries `workspace_id`. Stripe and Polar create billing management sessions for the selected workspace's provider customer. The app never exposes a local cross-workspace billing management surface.

A workspace with active external billing or a live checkout attempt cannot be deleted. This prevents a local deletion from leaving an external subscription charging without recovery data.

## Audit history

Member and role changes create tenant-visible audit events. So do invitation, project, and workspace changes. Events use redacted metadata. They never store tokens, secrets, or private provider URLs.

The global operator log is separate. See [Admin Panel](/docs/features/admin-panel).

## When not to select Workspaces

Leave Workspaces disabled when every account owns only its own data and subscription. The generator then omits all Workspaces code. Workspace records. Members. Invitations. Workspace switching. Workspace RBAC.
