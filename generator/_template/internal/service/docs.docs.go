package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"{{.ModulePath}}/internal/markdown"
	"{{.ModulePath}}/internal/model"
)

type DocsService struct {
	parser      *markdown.Parser
	contentPath string
	docsTree    *model.DocPage
}

func NewDocsService(contentPath string) *DocsService {
	return &DocsService{
		parser:      markdown.NewParser(),
		contentPath: contentPath,
	}
}

func (s *DocsService) BuildDocsTree() error {
	docsPath := filepath.Join(s.contentPath, "docs")
	s.docsTree = &model.DocPage{
		Title:    "Documentation",
		Slug:     "",
		Path:     "",
		Children: []*model.DocPage{},
	}

	// Track directory metadata from _index.md files
	dirMetadata := make(map[string]*model.DocPage)

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

func (s *DocsService) loadDocPage(fullPath, relPath string) (*model.DocPage, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	htmlContent, meta, err := s.parser.ParseWithFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// relPath already normalized to forward slashes
	slug := strings.TrimSuffix(relPath, ".md")

	page := &model.DocPage{
		Slug:        slug,
		Path:        relPath,
		HTMLContent: string(htmlContent),
		Content:     string(content),
		Headings:    s.parser.ExtractHeadings(content),
		Children:    []*model.DocPage{},
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

func (s *DocsService) insertPage(page *model.DocPage, relPath string, dirMetadata map[string]*model.DocPage) {
	parts := strings.Split(relPath, "/") // Already normalized to forward slashes
	current := s.docsTree

	// Build/traverse the directory structure
	for i := 0; i < len(parts)-1; i++ {
		dirSlug := strings.Join(parts[:i+1], "/")

		// Check if this directory already exists
		var found *model.DocPage
		for _, child := range current.Children {
			if child.Slug == dirSlug {
				found = child
				break
			}
		}

		if found == nil {
			// Create new directory node
			dirPage := &model.DocPage{
				Slug:     dirSlug,
				Path:     dirSlug,
				Children: []*model.DocPage{},
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

func (s *DocsService) sortTree(node *model.DocPage) {
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

func (s *DocsService) DocPage(slug string) (*model.DocPage, error) {
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

func (s *DocsService) findFirstContentPage(node *model.DocPage) *model.DocPage {
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

func (s *DocsService) findPage(node *model.DocPage, slug string) *model.DocPage {
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

func (s *DocsService) DocsTree() (*model.DocPage, error) {
	if s.docsTree == nil {
		err := s.BuildDocsTree()
		if err != nil {
			return nil, err
		}
	}
	return s.docsTree, nil
}

// FlatDocsList returns all documentation pages in a flat list, in the order they appear in the sidebar
func (s *DocsService) FlatDocsList() []*model.DocPage {
	if s.docsTree == nil {
		return []*model.DocPage{}
	}

	var pages []*model.DocPage
	s.collectPagesInOrder(s.docsTree, &pages)
	return pages
}

// collectPagesInOrder recursively collects all pages in sidebar order
func (s *DocsService) collectPagesInOrder(node *model.DocPage, pages *[]*model.DocPage) {
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
func (s *DocsService) PrevNextPages(currentPage *model.DocPage) (prev, next *model.DocPage) {
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
