package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"{{.ModulePath}}/internal/markdown"
	"{{.ModulePath}}/internal/model"
)

type BlogService struct {
	parser      *markdown.Parser
	contentPath string
}

func NewBlogService(contentPath string) *BlogService {
	return &BlogService{
		parser:      markdown.NewParser(),
		contentPath: contentPath,
	}
}

func (s *BlogService) Posts() ([]*model.BlogPost, error) {
	pattern := filepath.Join(s.contentPath, "blog", "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var posts []*model.BlogPost
	for _, file := range files {
		post, err := s.Post(filepath.Base(file[:len(file)-3]))
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	return posts, nil
}

func (s *BlogService) Post(slug string) (*model.BlogPost, error) {
	path := filepath.Join(s.contentPath, "blog", slug+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("blog post not found: %s", slug)
	}

	htmlContent, meta, err := s.parser.ParseWithFrontmatter(content)
	if err != nil {
		return nil, err
	}

	post := &model.BlogPost{
		Slug:        slug,
		HTMLContent: string(htmlContent),
		Content:     string(content),
	}

	if title, ok := meta["title"].(string); ok {
		post.Title = title
	}
	if author, ok := meta["author"].(string); ok {
		post.Author = author
	}
	if description, ok := meta["description"].(string); ok {
		post.Description = description
	}
	if dateStr, ok := meta["date"].(string); ok {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			post.Date = date
		}
	}
	if tags, ok := meta["tags"].([]any); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				post.Tags = append(post.Tags, tagStr)
			}
		}
	}
	if heroImage, ok := meta["hero_image"].(string); ok {
		post.HeroImage = heroImage
	}

	post.ReadTime = s.calculateReadTime(string(content))

	return post, nil
}

func (s *BlogService) PostsByTag(tag string) ([]*model.BlogPost, error) {
	allPosts, err := s.Posts()
	if err != nil {
		return nil, err
	}

	var posts []*model.BlogPost
	for _, post := range allPosts {
		for _, postTag := range post.Tags {
			if strings.EqualFold(postTag, tag) {
				posts = append(posts, post)
				break
			}
		}
	}

	return posts, nil
}

func (s *BlogService) calculateReadTime(content string) int {
	words := strings.Fields(content)
	wordsPerMinute := 200
	readTime := len(words) / wordsPerMinute
	if readTime < 1 {
		readTime = 1
	}
	return readTime
}
