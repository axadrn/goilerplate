# goilerplate

The production Go stack you own.

The CLI for generating production-ready Go SaaS projects.

## Install

```bash
go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest
```

## Usage

```bash
goilerplate new [--edition free|paid] [--framework htmx|datastar|svelte|headless] [--api] [--headless] [--mcp] --module example.com/acme ./acme
```

Run `goilerplate new` without options in a terminal for interactive setup. `--api` adds the JSON API to an htmx or Datastar project. `--framework svelte` generates a SvelteKit app on the JSON API, served by the Go binary. It includes the JSON API and needs Node.js 22.17 or newer and pnpm. `--headless` generates the Go backend with the JSON API and no native frontend. Both work in Free and Paid. `--mcp` adds an MCP server so your users can connect their AI agents with a personal API token. It needs the JSON API.

Run `goilerplate doctor` inside a generated project to check its tools and `.env`. For SvelteKit it checks Node.js and pnpm.

## Documentation

Visit [goilerplate.com/docs](https://goilerplate.com/docs).

## License

Licensed under the [MIT License](LICENSE).
