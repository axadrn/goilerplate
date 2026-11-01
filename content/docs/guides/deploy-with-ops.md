---
title: "Deploy with ops"
description: "Your goilerplate app live on your own server. HTTPS and SQLite backups included."
order: 2
---

## TLDR

[ops](https://ops.goilerplate.com/?ref=goilerplate-guide) deploys your goilerplate app to your own server and backs up its SQLite database. Same maker, made for this stack.

```bash
ops servers add web-1
ops apps add --server web-1 acme
ops env set APP_URL=https://acme.com SMTP_HOST=smtp.acme.com ...
ops deploy
```

Your app is live with HTTPS on your domain. Every change of its database is backed up to your bucket. You set no database path: the `compose.yml` goilerplate wrote already keeps it.

## ELI5

You rent a small Linux server, for example for $5 a month. ops installs a small agent on it. The agent builds your app from its Dockerfile, starts it and keeps watching it.

Your database stays a plain SQLite file on that server. ops copies every change of it to your own bucket. Delete the wrong rows? Go back to the second before.

## What you get

- Your app on your domain, with HTTPS.
- Every change of the SQLite database backed up to your bucket, and a restore test every day.
- Restore to any second of the last week, or to the snapshot before a deploy.
- A data browser: tables, rows and SQL, in the browser, the CLI, a terminal UI and MCP.
- Deploy logs, app logs and roll back in one click.

Postgres projects deploy too. Backups, restore and the data browser are for SQLite.

## 1. Install the CLI

On a Mac or Linux:

```bash
curl -fsSL https://ops.goilerplate.com/cli.sh | sh
```

## 2. Log in

[Start the free trial](https://ops.goilerplate.com/auth?ref=goilerplate-guide): 30 days, no card. Create a personal API token under **Settings → Tokens**, then:

```bash
ops login
```

Paste the token when it asks.

## 3. Add your server

Any Linux server with root access works, at any provider.

```bash
ops servers add web-1
```

It prints one command. Run it on the server as root. `ops servers` then shows `web-1 online`.

## 4. Add your bucket

Your code and your backups go into your own S3 bucket: Cloudflare R2, AWS, Backblaze or Hetzner.

```bash
ops backups bucket set --endpoint https://<account>.r2.cloudflarestorage.com --bucket my-ops --access-key <key>
```

## 5. Set up the app

Run the next commands in your project folder. `acme` is the name of your project.

```bash
ops apps add --server web-1 acme
```

Set the production values from your `.env.example`:

```bash
ops env set \
  APP_URL=https://acme.com \
  EMAIL_FROM=noreply@acme.com \
  RESEND_API_KEY=re_... \
  ADMIN_EMAILS=you@acme.com
```

Add the payment, OAuth and storage keys your stack uses. The app lists every missing value at startup.

Leave `DB_CONNECTION` out. The database stays in `data/`. Your project's `compose.yml` mounts that folder, so ops keeps it across deploys and backs it up.

## 6. Add your domain

```bash
ops domains add acme.com
```

Point an A record of `acme.com` at the server's IP. ops shows the address and the record to set. HTTPS comes by itself.

## 7. Deploy

```bash
ops deploy
```

ops sends your code to your bucket. The server builds the Dockerfile goilerplate wrote, waits until `/healthz` answers and puts the app live. The log names the database it found in `/app/data` and ends with `v1 is live`.

Every next deploy is the same one command. ops takes a snapshot of the database before each one.

## Real visitor IPs

The sign-in rate limit counts requests per visitor IP. Your app sits behind the proxy ops runs, so every request seems to come from the proxy. Without this step, all your visitors share one limit. Tell the app to trust the proxy's network, once. On the server:

```bash
docker network inspect ops-proxy --format '{{(index .IPAM.Config 0).Subnet}}'
```

Then set the subnet it prints, and restart:

```bash
ops env set TRUSTED_PROXY_CIDRS=172.20.0.0/16
ops restart
```

## Next

- [Getting started](https://ops.goilerplate.com/docs/getting-started?ref=goilerplate-guide) on ops.
- [Backups and restore](https://ops.goilerplate.com/docs/guides/backups?ref=goilerplate-guide).
- [Your data](https://ops.goilerplate.com/docs/guides/data?ref=goilerplate-guide): tables, rows and SQL.
- [Push to deploy](https://ops.goilerplate.com/docs/guides/deploy?ref=goilerplate-guide#push-to-deploy-with-github-actions) with GitHub Actions.
- [Deployment](/docs/guides/deployment) for any other host.
