package model

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

type DocHeading struct {
	ID    string
	Text  string
	Level int
}
