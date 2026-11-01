package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/v4/site/markdown"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type LegalPage struct {
	Title       string
	Slug        string
	Content     string
	LastUpdated string
}

type LegalService struct {
	contentDir   string
	supportEmail string
	pages        map[string]*LegalPage
}

// NewLegalService serves the pages in <contentDir>/legal. {{support_email}}
// in a page becomes supportEmail.
func NewLegalService(contentDir, supportEmail string) *LegalService {
	return &LegalService{
		contentDir:   filepath.Join(contentDir, "legal"),
		supportEmail: supportEmail,
		pages:        make(map[string]*LegalPage),
	}
}

func (s *LegalService) LoadPages() error {
	files, err := os.ReadDir(s.contentDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if it doesn't exist
			err = os.MkdirAll(s.contentDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create legal directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to read legal directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		slug := strings.TrimSuffix(file.Name(), ".md")
		page, err := s.loadPage(slug)
		if err != nil {
			return fmt.Errorf("failed to load page %s: %w", slug, err)
		}

		s.pages[slug] = page
	}

	return nil
}

func (s *LegalService) loadPage(slug string) (*LegalPage, error) {
	filePath := filepath.Join(s.contentDir, slug+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content = []byte(strings.ReplaceAll(string(content), "{{support_email}}", s.supportEmail))
	parser := markdown.NewParser()
	html, meta, err := parser.ParseWithFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Get file info for last updated date
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	title, _ := meta["title"].(string)
	if title == "" {
		// Generate title from slug
		title = cases.Title(language.English).String(strings.ReplaceAll(slug, "-", " "))
	}

	// Get lastUpdated from frontmatter first, fallback to file modification time
	var lastUpdated string
	dateValue, ok := meta["lastUpdated"]
	if ok {
		// Try to parse various date formats
		lastUpdated = s.parseDate(dateValue)
	}

	// Fallback to file modification time if not in frontmatter
	if lastUpdated == "" {
		lastUpdated = info.ModTime().Format("January 2, 2006")
	}

	return &LegalPage{
		Title:       title,
		Slug:        slug,
		Content:     string(html),
		LastUpdated: lastUpdated,
	}, nil
}

func (s *LegalService) Page(slug string) (*LegalPage, error) {
	// Reload to get latest content in development
	err := s.LoadPages()
	if err != nil {
		return nil, err
	}

	page, ok := s.pages[slug]
	if !ok {
		return nil, fmt.Errorf("page not found: %s", slug)
	}

	return page, nil
}

// parseDate tries to parse various date formats and returns formatted date
func (s *LegalService) parseDate(value interface{}) string {
	var dateStr string

	switch v := value.(type) {
	case string:
		dateStr = v
	case time.Time:
		return v.Format("January 2, 2006")
	default:
		return ""
	}

	// Try various date formats
	formats := []string{
		"2006-01-02",      // ISO date
		"2006/01/02",      // Alternative
		"02.01.2006",      // European
		"01/02/2006",      // US format
		"Jan 2, 2006",     // Short month
		"January 2, 2006", // Full month
		time.RFC3339,      // RFC3339
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t.Format("January 2, 2006")
		}
	}

	// Return as-is if parsing fails
	return dateStr
}
