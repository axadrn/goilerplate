package docs

import "github.com/axadrn/goilerplate/v4/site/markdown"

type DocPage struct {
	Title       string
	Slug        string
	Path        string
	Order       int
	Description string
	Content     string
	HTMLContent string
	Headings    []DocHeading
	Children    []*DocPage
	Parent      *DocPage
}

// DocHeading is a heading of a page, as the table of contents shows it.
type DocHeading = markdown.Heading
