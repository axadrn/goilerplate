// Package sitectx carries what a page needs to know about the build into its
// templates: the path it is rendered for and the numbers fetched at build
// time.
package sitectx

import "context"

type key struct{}

// Page is the build context of one rendered page.
type Page struct {
	Path  string
	Stars int
}

// With returns a context for rendering the page.
func With(ctx context.Context, page Page) context.Context {
	return context.WithValue(ctx, key{}, page)
}

func from(ctx context.Context) Page {
	page, _ := ctx.Value(key{}).(Page)
	return page
}

// URLPath is the path the page is rendered for, such as /docs/installation.
func URLPath(ctx context.Context) string { return from(ctx).Path }

// Stars is the number of GitHub stars of the repo at build time.
func Stars(ctx context.Context) int { return from(ctx).Stars }
