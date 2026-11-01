---
title: "MCP server"
description: "Let your users connect Claude, Cursor, or VS Code to your SaaS. A curated MCP tool list on the same services, behind personal API tokens."
order: 2
---

## TLDR

Add `--mcp` next to `--api`, `--framework svelte`, or `--headless`:

```bash
goilerplate new --module example.com/acme --framework htmx --api --mcp ./acme
goilerplate new --module example.com/acme --headless --mcp ./acme
```

Your project then serves an [MCP](https://modelcontextprotocol.io) server at `/mcp`. Your users create a [personal API token](/docs/features/api-tokens), paste one command or config block into their agent, and ask it about their account, workspace, members, activity, or billing.

Your modules decide which tools exist, never the transport.

## How it works

MCP is a third transport next to the HTML handlers and the [JSON API](/docs/features/json-api). Tools live in `internal/mcp` and call the same services, so every rule and permission check still exists once.

- Streamable HTTP at `POST /mcp`, stateless. Every request is one complete exchange, so it scales like the JSON API. `GET` answers 405.
- Built on the official Go SDK, `github.com/modelcontextprotocol/go-sdk`.
- Only personal API tokens authenticate: `Authorization: Bearer pat_...`. Sessions, cookies, and unknown, revoked, or expired tokens get 401 `unauthenticated` with `WWW-Authenticate: Bearer realm="mcp"`.
- Every tool has typed input and output schemas. Results carry structured content and a text summary, so clients that only read text still get the data.
- A failed call is an error result whose text is the JSON API envelope with the same codes, for example `insufficient_scope`, `not_found`, `validation_failed`, or `seat_limit`.

The server sends instructions to the agent when it connects: whose account the tools act for, what the scopes mean, and how errors look.

## Tools

Agents get a short, curated list, not a copy of every API route.

| Tool | Scope | Selection | Does |
| --- | --- | --- | --- |
| `whoami` | read | Every project | The user, the token's name, scopes, and expiry. With workspaces, the bound workspace and the user's role there. |
| `get_billing` | read | Every project | Plan, subscription status, access state, and the plans that can be bought. Never a checkout or portal link. Owners only with workspaces. |
| `list_workspaces` | read | Workspaces | The workspaces the token reaches, with the user's role. |
| `get_workspace` | read | Workspaces | Name, slug, role and permissions, members, pending invitations, and seats. |
| `list_activity` | read | Workspaces | The audit log, newest first, cursor paginated. Owners and admins. |
| `invite_member` | write | Workspaces | Invites an email as member or admin and sends the invitation. The seat limit applies. |
| `rename_workspace` | write | Workspaces | Renames the workspace. The slug stays. |

Workspace tools take an optional `workspace` slug and default to the token's workspace. Every other slug answers `not_found`, including other workspaces of the same user. Roles still apply: a member's token cannot rename the workspace.

These are never tools: checkout, billing management, account or workspace deletion, member role changes, member removal, admin, impersonation, passwords, and email. An agent can read your billing state, but it cannot spend money or remove people.

## Scopes

A token has the `read` scope, or `read` and `write`. Read tools work with every token. Write tools need `write`; a read token gets `insufficient_scope` and nothing changes. Give agents a read token unless they need to act.

## Connect a client

Every client needs the public URL of your server with `/mcp`, for example `https://app.example.com/mcp`, and a token from Settings, or from `POST /api/v1/account/tokens` in a Headless project. Replace `app` with any name you like.

### Claude Code

```bash
claude mcp add --transport http app https://app.example.com/mcp \
  --header "Authorization: Bearer pat_..."
```

`claude mcp add` stores the server for the current project. Add `--scope user` to use it everywhere.

### Claude Desktop and claude.ai

Open Customize, then Connectors, then Add custom connector. Enter a name and the `/mcp` URL. Under Authentication choose "No sign in", and under Request headers add `Authorization` with the value `Bearer pat_...`. Claude connects from Anthropic's cloud, so the URL must be reachable from the internet. Request headers on custom connectors are in beta and not available to every organization yet. If you do not see the field, use the local bridge below.

For a server on your own machine, or when your organization has no Request headers field, Claude Desktop can start a local bridge instead. Add it to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "app": {
      "command": "npx",
      "args": ["mcp-remote", "http://localhost:8090/mcp", "--header", "Authorization:${AUTH_HEADER}"],
      "env": {"AUTH_HEADER": "Bearer pat_..."}
    }
  }
}
```

The header has no space after the colon and the token sits in `env`, because some clients do not quote spaces inside `args`.

### Cursor

`.cursor/mcp.json` in the project, or `~/.cursor/mcp.json` for every project, with the token in the `API_TOKEN` environment variable:

```json
{
  "mcpServers": {
    "app": {
      "url": "https://app.example.com/mcp",
      "headers": {"Authorization": "Bearer ${env:API_TOKEN}"}
    }
  }
}
```

### VS Code

`.vscode/mcp.json`. VS Code asks for the token the first time and stores it:

```json
{
  "inputs": [
    {"type": "promptString", "id": "api-token", "description": "Personal API token", "password": true}
  ],
  "servers": {
    "app": {
      "type": "http",
      "url": "https://app.example.com/mcp",
      "headers": {"Authorization": "Bearer ${input:api-token}"}
    }
  }
}
```

### One-click connect

Clients send the token header. The server answers 401 with `WWW-Authenticate: Bearer` without one, and some clients then offer an OAuth sign-in that this server does not provide yet. Check the header when you see that prompt. OAuth 2.1 one-click connect is planned for a later release.

## Try it

These ran against fresh local builds. Your IDs and tokens will differ.

### MCP Inspector

The [MCP Inspector](https://github.com/modelcontextprotocol/inspector) CLI against an htmx project with a read token:

```bash
npx @modelcontextprotocol/inspector --cli http://localhost:8090/mcp \
  --transport http --header "Authorization: Bearer $TOKEN" \
  --method tools/call --tool-name whoami
```

```json
{
  "content": [
    {
      "type": "text",
      "text": "Signed in as Ada Lovelace (ada@example.com) with the token \"Reader\" and the scopes read."
    },
    {
      "type": "text",
      "text": "{\"user\":{\"id\":\"523fec91-7a14-43e7-be19-a3b32f8941f8\",\"email\":\"ada@example.com\",\"name\":\"Ada Lovelace\",\"password_set\":false,\"created_at\":\"2026-10-05T13:08:51.679317704Z\"},\"token\":{\"name\":\"Reader\",\"scopes\":[\"read\"],\"expires_at\":\"2026-11-04T13:08:51.696117714Z\"}}"
    }
  ],
  "structuredContent": {
    "token": {
      "expires_at": "2026-11-04T13:08:51.696117714Z",
      "name": "Reader",
      "scopes": [
        "read"
      ]
    },
    "user": {
      "created_at": "2026-10-05T13:08:51.679317704Z",
      "email": "ada@example.com",
      "id": "523fec91-7a14-43e7-be19-a3b32f8941f8",
      "name": "Ada Lovelace",
      "password_set": false
    }
  }
}
```

`--method tools/list` on a project without workspaces returns `whoami` and `get_billing`. On a Headless workspaces project it returns seven tools. The first text content of each call, with the owner's read token and then the write token:

```text
$ tools/list
get_billing, get_workspace, invite_member, list_activity, list_workspaces, rename_workspace, whoami

$ get_workspace
Analytical Engines (slug ada-lovelace-s-workspace-0170a818): 1 of 25 seats used, 0 pending invitations, your role is owner.

$ get_workspace workspace=someone-else
{"error":{"code":"not_found","message":"The workspace was not found."}}

$ invite_member email=linus@example.com role=member   (read token)
{"error":{"code":"insufficient_scope","message":"This API token needs the write scope. Use a token with read and write access."}}

$ invite_member email=linus@example.com role=member   (write token)
Invited linus@example.com to Analytical Engines as member. The invitation email is on its way.

$ get_billing
Plan enterprise, access state paid.

$ list_activity limit=2
2 events, newest first. Pass next_cursor as cursor for older events.
```

### Claude Code

The same workspaces project, connected with the read token:

```bash
claude mcp add --transport http app http://localhost:8090/mcp \
  --header "Authorization: Bearer $TOKEN"
claude mcp get app
```

```text
app:
  Scope: Local config (private to you in this project)
  Status: ✔ Connected
  Type: http
  URL: http://localhost:8090/mcp
  Headers:
    Authorization: Bearer pat_...
```

```bash
claude -p "Which workspace am I in, how many seats are used, and are there pending invitations? Then rename the workspace to Difference Engine." \
  --allowedTools "mcp__app__*"
```

Claude called `get_workspace`, then `rename_workspace`, which failed with `insufficient_scope`. Its answer, without the Markdown formatting:

```text
I got the workspace details, but the rename failed.

Workspace: Analytical Engines (slug ada-lovelace-s-workspace-0170a818). You're the owner.

Seats: 1 of 25 used. The only member is Ada Lovelace (ada@example.com).

Pending invitations: 1. linus@example.com was invited as a member; the invitation expires 2026-10-12.

Rename: The workspace is still called "Analytical Engines." The server rejected the request with insufficient_scope because your API token only has read access. As owner, you're allowed to rename it. To fix this, create a token with read and write access in account settings and update the app MCP server to use it. Then I can rename it to "Difference Engine." You could also rename it yourself in the app.
```

The workspace name in the database stayed Analytical Engines.

## Add your own tool

The built-in tools are a starting point. Your product's own data is what users want their agents to reach. Add a tool next to its service: wrap the body in `mcp.Read` or `mcp.Write`, call the service, and return a struct. The SDK derives the input and output schemas from the types.

This example ships in every MCP project as `internal/mcp/example_test.go`, where a test runs it against a real server and database. Inside `internal/mcp`, drop the `mcp.` prefixes and call it from `registerTools` in `internal/mcp/tools.go`:

```go
type nameInput struct {
	Name string `json:"name" jsonschema:"the new display name"`
}

func registerNameTool(server *mcpsdk.Server, services mcp.Services) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "set_display_name",
		Description: "Changes the display name of the user this token acts for. Needs a token with read and write access.",
	}, mcp.Write(func(ctx context.Context, caller mcp.Caller, input nameInput) (api.UserResponse, error) {
		err := validation.ValidateName(input.Name)
		if err != nil {
			return api.UserResponse{}, mcp.InvalidField("name", api.FieldMessage(err))
		}
		user, err := services.Users.UpdateName(caller.User.ID, input.Name)
		if err != nil {
			return api.UserResponse{}, err
		}
		return api.UserResponse{User: api.NewUser(user)}, nil
	}))
}
```

- `mcp.Write` checks the `write` scope before the body runs. `mcp.Read` is for tools that only read.
- `caller.User` and `caller.Token` are the user and the token of the request.
- Return service errors unchanged. They reach the agent with the JSON API code.
- `mcp.InvalidField` reports a `validation_failed` field like the API does.
- The description is what the agent reads. Say what the tool returns or changes and which scope it needs.

Tools call services through `mcp.Services` in `internal/mcp/server.go`. Add your service there and pass it where `internal/routes/routes.go` calls `mcp.NewServer`. Raise `mcp.Version` when tools change.

The generated `AGENTS.md` carries the same rules, so a coding agent adds tools the same way.

## Production

`/mcp` runs in the same binary and behind the same middleware as the JSON API: request logs, security headers, and the token checks. `invite_member` shares the invitation limit of 30 an hour per member and workspace. It sets no cookies and needs no CSRF token. Behind a reverse proxy, forward `/mcp` like `/api/`.
