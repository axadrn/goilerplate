package docs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/axadrn/goilerplate/v4/site/markdown"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type DocsService struct {
	parser       *markdown.Parser
	contentPath  string
	supportEmail string
	docsTree     *DocPage
}

// NewDocsService reads the pages in <contentPath>/docs. {{support_email}} in
// a page becomes supportEmail.
func NewDocsService(contentPath, supportEmail string) *DocsService {
	return &DocsService{
		parser:       markdown.NewParser(),
		contentPath:  contentPath,
		supportEmail: supportEmail,
	}
}

func (s *DocsService) BuildDocsTree() error {
	docsPath := filepath.Join(s.contentPath, "docs")
	s.docsTree = &DocPage{
		Title:    "Documentation",
		Slug:     "",
		Path:     "",
		Children: []*DocPage{},
	}

	// Track directory metadata from _index.md files
	dirMetadata := make(map[string]*DocPage)

	err := filepath.Walk(docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}

		relPath, err := filepath.Rel(docsPath, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		page, err := s.loadDocPage(path, relPath)
		if err != nil {
			return err
		}

		// Handle _index.md specially. Store metadata for the directory.
		if strings.HasSuffix(relPath, "_index.md") {
			dir := strings.TrimSuffix(relPath, "/_index.md")
			if dir == "_index.md" { // root _index.md
				dir = ""
			}
			dirMetadata[dir] = page
			return nil
		}

		// Insert regular pages into tree
		s.insertPage(page, relPath, dirMetadata)
		return nil
	})

	if err != nil {
		return err
	}

	s.sortTree(s.docsTree)
	return nil
}

func (s *DocsService) loadDocPage(fullPath, relPath string) (*DocPage, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	content = bytes.ReplaceAll(content, []byte("{{support_email}}"), []byte(s.supportEmail))

	htmlContent, meta, err := s.parser.ParseWithFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// relPath already normalized to forward slashes
	slug := strings.TrimSuffix(relPath, ".md")

	page := &DocPage{
		Slug:        slug,
		Path:        relPath,
		HTMLContent: string(htmlContent),
		Content:     string(content),
		Headings:    s.parser.ExtractHeadings(content),
		Children:    []*DocPage{},
	}

	title, ok := meta["title"].(string)
	if ok {
		page.Title = title
	} else {
		page.Title = s.titleFromSlug(slug)
	}

	description, ok := meta["description"].(string)
	if ok {
		page.Description = description
	}

	order, ok := meta["order"].(int)
	if ok {
		page.Order = order
	} else {
		orderFloat, ok := meta["order"].(float64)
		if ok {
			page.Order = int(orderFloat)
		}
	}

	return page, nil
}

func (s *DocsService) insertPage(page *DocPage, relPath string, dirMetadata map[string]*DocPage) {
	parts := strings.Split(relPath, "/") // Already normalized to forward slashes
	current := s.docsTree

	// Build/traverse the directory structure
	for i := 0; i < len(parts)-1; i++ {
		dirSlug := strings.Join(parts[:i+1], "/")

		// Check if this directory already exists
		var found *DocPage
		for _, child := range current.Children {
			if child.Slug == dirSlug {
				found = child
				break
			}
		}

		if found == nil {
			// Create new directory node
			dirPage := &DocPage{
				Slug:     dirSlug,
				Path:     dirSlug,
				Children: []*DocPage{},
				Parent:   current,
			}

			// Apply metadata from _index.md if available
			meta, ok := dirMetadata[dirSlug]
			if ok {
				dirPage.Title = meta.Title
				dirPage.Description = meta.Description
				dirPage.Order = meta.Order
				dirPage.HTMLContent = meta.HTMLContent
				dirPage.Content = meta.Content
				dirPage.Headings = meta.Headings
			} else {
				// Generate title from slug
				dirPage.Title = s.titleFromSlug(parts[i])
			}

			current.Children = append(current.Children, dirPage)
			current = dirPage
		} else {
			current = found
		}
	}

	// Add the page to its parent directory
	page.Parent = current
	current.Children = append(current.Children, page)
}

func (s *DocsService) sortTree(node *DocPage) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Order != node.Children[j].Order {
			return node.Children[i].Order < node.Children[j].Order
		}
		return node.Children[i].Title < node.Children[j].Title
	})

	for _, child := range node.Children {
		s.sortTree(child)
	}
}

func (s *DocsService) DocPage(slug string) (*DocPage, error) {
	if s.docsTree == nil {
		err := s.BuildDocsTree()
		if err != nil {
			return nil, err
		}
	}

	if slug == "" {
		// Find the first actual content page (not a directory)
		firstPage := s.findFirstContentPage(s.docsTree)
		if firstPage != nil {
			return firstPage, nil
		}
		// Fall back to tree root if no content pages
		return s.docsTree, nil
	}

	page := s.findPage(s.docsTree, slug)
	if page == nil {
		return nil, fmt.Errorf("documentation page not found: %s", slug)
	}

	return page, nil
}

func (s *DocsService) findFirstContentPage(node *DocPage) *DocPage {
	// Check children in order
	for _, child := range node.Children {
		// If this child has no children, it's a content page
		if len(child.Children) == 0 {
			return child
		}
		// Otherwise, recursively check this child's children
		page := s.findFirstContentPage(child)
		if page != nil {
			return page
		}
	}
	return nil
}

func (s *DocsService) findPage(node *DocPage, slug string) *DocPage {
	if node.Slug == slug {
		return node
	}

	for _, child := range node.Children {
		found := s.findPage(child, slug)
		if found != nil {
			return found
		}
	}

	return nil
}

func (s *DocsService) DocsTree() (*DocPage, error) {
	if s.docsTree == nil {
		err := s.BuildDocsTree()
		if err != nil {
			return nil, err
		}
	}
	return s.docsTree, nil
}

// FlatDocsList returns all documentation pages in a flat list, in the order they appear in the sidebar
func (s *DocsService) FlatDocsList() []*DocPage {
	if s.docsTree == nil {
		return []*DocPage{}
	}

	var pages []*DocPage
	s.collectPagesInOrder(s.docsTree, &pages)
	return pages
}

// collectPagesInOrder recursively collects all pages in sidebar order
func (s *DocsService) collectPagesInOrder(node *DocPage, pages *[]*DocPage) {
	// Don't add the root node or category pages (nodes with children are categories)
	if node.Slug != "" && len(node.Children) == 0 {
		*pages = append(*pages, node)
	}

	// Add all children recursively
	for _, child := range node.Children {
		s.collectPagesInOrder(child, pages)
	}
}

// PrevNextPages returns the previous and next pages in the documentation flow
func (s *DocsService) PrevNextPages(currentPage *DocPage) (prev, next *DocPage) {
	pages := s.FlatDocsList()

	for i, page := range pages {
		if page.Slug == currentPage.Slug {
			if i > 0 {
				prev = pages[i-1]
			}
			if i < len(pages)-1 {
				next = pages[i+1]
			}
			break
		}
	}

	return prev, next
}

func (s *DocsService) titleFromSlug(slug string) string {
	parts := strings.Split(slug, "/")
	lastPart := parts[len(parts)-1]

	lastPart = strings.ReplaceAll(lastPart, "-", " ")
	lastPart = strings.ReplaceAll(lastPart, "_", " ")

	words := strings.Fields(lastPart)
	caser := cases.Title(language.English)
	for i, word := range words {
		words[i] = caser.String(word)
	}

	return strings.Join(words, " ")
}
