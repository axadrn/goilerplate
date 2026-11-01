// Package web embeds the built SvelteKit app. pnpm build writes it to
// web/build; web/build/.gitkeep keeps the directory, so the Go build works
// before the first pnpm build. pnpm build empties web/build and copies
// web/static/.gitkeep back.
package web

import "embed"

// Build is the static SvelteKit build. internal/routes serves it under /.
//
//go:embed all:build
var Build embed.FS
