# goilerplate

The production Go stack you own.

The CLI for generating production-ready Go SaaS projects.

## Install

```bash
go install github.com/axadrn/goilerplate/v3/cmd/goilerplate@latest
```

## Usage

```bash
goilerplate new [--edition free|paid] [--framework htmx|datastar|headless] [--api] [--headless] --module example.com/acme ./acme
```

Run `goilerplate new` without options in a terminal for interactive setup. `--api` adds the JSON API to an htmx or Datastar project. `--headless` generates the Go backend with the JSON API and no native frontend, in Free and Paid.

## Documentation

Visit [goilerplate.com/docs](https://goilerplate.com/docs).

## License

Licensed under the [MIT License](LICENSE).
