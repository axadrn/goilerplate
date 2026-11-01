// Command build renders the goilerplate website into a directory of static
// files: every page as HTML, the docs also as Markdown, the assets, the
// sitemap, robots.txt and llms.txt. Run it from the repository root after
// templ generate and the Tailwind build.
package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/axadrn/goilerplate/v4/site/brand"
	"github.com/axadrn/goilerplate/v4/site/docs"
	"github.com/axadrn/goilerplate/v4/site/githubstars"
	"github.com/axadrn/goilerplate/v4/site/sitectx"
	"github.com/axadrn/goilerplate/v4/site/ui/pages"
)

func main() {
	out := flag.String("out", "site/dist", "directory to write the site into")
	content := flag.String("content", "content", "directory with the docs and legal Markdown")
	changelog := flag.String("changelog", "CHANGELOG.md", "the changelog to publish")
	assets := flag.String("assets", "site/assets", "directory with the built assets")
	stars := flag.Bool("stars", true, "ask GitHub for the star count of the repo")
	flag.Parse()

	site := builder{out: *out}
	if *stars {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		count, err := githubstars.Fetch(ctx, &http.Client{})
		cancel()
		if err != nil {
			log.Printf("GitHub stars unavailable, the navbar shows none: %v", err)
		}
		site.stars = count
	}
	if err := site.build(*content, *changelog, *assets); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d files to %s", site.files, site.out)
}

type builder struct {
	out   string
	stars int
	files int
	paths []string
}

func (b *builder) build(content, changelog, assets string) error {
	if err := os.RemoveAll(b.out); err != nil {
		return err
	}
	if err := copyDir(assets, filepath.Join(b.out, "assets"), b); err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	if err := b.page("/", pages.Landing()); err != nil {
		return err
	}
	if err := b.page("/new", pages.Configurator()); err != nil {
		return err
	}
	if err := b.changelog(changelog); err != nil {
		return err
	}
	if err := b.docs(content); err != nil {
		return err
	}
	if err := b.legal(content); err != nil {
		return err
	}
	if err := b.render("404.html", "/404", pages.NotFound()); err != nil {
		return err
	}
	if err := b.write("robots.txt", []byte("User-agent: *\nAllow: /\nSitemap: "+brand.URL+"/sitemap.xml\n")); err != nil {
		return err
	}
	return b.sitemap()
}

// page renders a page that belongs in the sitemap.
func (b *builder) page(path string, component templ.Component) error {
	b.paths = append(b.paths, path)
	return b.render(htmlFile(path), path, component)
}

func (b *builder) render(name, path string, component templ.Component) error {
	var buffer bytes.Buffer
	ctx := sitectx.With(context.Background(), sitectx.Page{Path: path, Stars: b.stars})
	if err := component.Render(ctx, &buffer); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	return b.write(name, buffer.Bytes())
}

func (b *builder) write(name string, content []byte) error {
	target := filepath.Join(b.out, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	b.files++
	return os.WriteFile(target, content, 0o644)
}

// htmlFile is where the page of path lives: / is index.html, /docs/x is
// docs/x.html. The server answers /docs/x with it.
func htmlFile(path string) string {
	if path == "/" {
		return "index.html"
	}
	return strings.TrimPrefix(path, "/") + ".html"
}

func (b *builder) changelog(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read changelog: %w", err)
	}
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(html.WithUnsafe()))
	var rendered bytes.Buffer
	if err := markdown.Convert(source, &rendered); err != nil {
		return fmt.Errorf("render changelog: %w", err)
	}
	return b.page("/changelog", pages.Changelog(rendered.String()))
}

func (b *builder) docs(content string) error {
	service := docs.NewDocsService(content, brand.SupportEmail)
	tree, err := service.DocsTree()
	if err != nil {
		return fmt.Errorf("read docs: %w", err)
	}
	var llms strings.Builder
	fmt.Fprintf(&llms, "# %s\n\n> %s. One command writes a production Go SaaS onto your machine: login, billing, teams, admin, an API and tests.\n\n## Documentation\n\n", brand.Name, brand.Tagline)
	for _, page := range service.FlatDocsList() {
		path := docsPath(page)
		prev, next := service.PrevNextPages(page)
		if err := b.page(path, pages.Docs(page, tree, prev, next)); err != nil {
			return err
		}
		if err := b.write(strings.TrimPrefix(path, "/")+".md", []byte(page.Content)); err != nil {
			return err
		}
		description := strings.TrimSpace(page.Description)
		if description == "" {
			description = "goilerplate documentation"
		}
		fmt.Fprintf(&llms, "- [%s](%s%s.md): %s\n", page.Title, brand.URL, path, description)
	}
	fmt.Fprintf(&llms, "\n## Product\n\n- [Stack builder](%s/new): Choose the stack and copy the matching goilerplate new command.\n- [Changelog](%s/changelog): What shipped, newest first.\n- [Source](https://github.com/%s): The CLI, the generator and this website.\n", brand.URL, brand.URL, githubstars.Repo)
	return b.write("llms.txt", []byte(llms.String()))
}

// docsPath is the URL of a docs page; the introduction is /docs.
func docsPath(page *docs.DocPage) string {
	if page.Slug == "" || page.Slug == "introduction" {
		return "/docs"
	}
	return "/docs/" + page.Slug
}

func (b *builder) legal(content string) error {
	service := docs.NewLegalService(content, brand.SupportEmail)
	if err := service.LoadPages(); err != nil {
		return fmt.Errorf("read legal pages: %w", err)
	}
	page, err := service.Page("privacy")
	if err != nil {
		return fmt.Errorf("read privacy policy: %w", err)
	}
	return b.page("/legal/privacy", pages.Legal(page))
}

func (b *builder) sitemap() error {
	type url struct {
		Loc string `xml:"loc"`
	}
	type urlset struct {
		XMLName xml.Name `xml:"urlset"`
		XMLNS   string   `xml:"xmlns,attr"`
		URLs    []url    `xml:"url"`
	}
	set := urlset{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, path := range b.paths {
		set.URLs = append(set.URLs, url{Loc: brand.URL + strings.TrimSuffix(path, "/")})
	}
	content, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	return b.write("sitemap.xml", append([]byte(xml.Header), append(content, '\n')...))
}

// copyDir copies the assets the pages use. The Tailwind input files stay
// behind; output.css is what the pages load.
func copyDir(source, target string, b *builder) error {
	if _, err := os.Stat(filepath.Join(source, "css", "output.css")); err != nil {
		return errors.New("site/assets/css/output.css is missing: build the CSS with Tailwind first")
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if strings.HasPrefix(name, "css/") && name != "css/output.css" {
			return nil
		}
		if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(b.out, filepath.Join(target, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		return b.write(filepath.ToSlash(relative), content)
	})
}
