// Command server is the goilerplate website: it renders the templ pages on
// every request, serves the assets, the docs also as Markdown, the sitemap,
// robots.txt and llms.txt, redirects old URLs, and gives CLIs from before v4,
// which still call the retired API, a plain answer. Run it from the
// repository root after templ generate and the Tailwind build.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	addr := flag.String("addr", ":8090", "address to listen on")
	content := flag.String("content", "content", "directory with the docs and legal Markdown")
	changelog := flag.String("changelog", "CHANGELOG.md", "the changelog to publish")
	assets := flag.String("assets", "site/assets", "directory with the built assets")
	stars := flag.Bool("stars", true, "ask GitHub for the star count of the repo every hour")
	dev := flag.Bool("dev", false, "read the Markdown again on every request and leave out the analytics")
	flag.Parse()

	site, err := newSite(*content, *changelog, *assets, *dev)
	if err != nil {
		log.Fatal(err)
	}
	if *stars {
		go site.fetchStars()
	}
	server := &http.Server{
		Addr:              *addr,
		Handler:           site,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("serving the site on %s", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func healthcheck() int {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:8090/healthz")
	if err != nil {
		return 1
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return 1
	}
	return 0
}

// redirects sends the URLs of the paid site and moved docs somewhere useful.
var redirects = map[string]string{
	"/pricing":                        "/",
	"/license":                        "/",
	"/affiliates":                     "/",
	"/legal/terms":                    "/legal/privacy",
	"/ops":                            "https://moin.run",
	"/moin":                           "https://moin.run",
	"/blog":                           "/",
	"/docs/introduction":              "/docs",
	"/docs/introduction.md":           "/docs.md",
	"/docs/guides/cli-accounts":       "/docs/installation",
	"/docs/guides/cli-accounts.md":    "/docs/installation.md",
	"/docs/guides/deploy-with-ops":    "/docs/guides/deploy-with-moin",
	"/docs/guides/deploy-with-ops.md": "/docs/guides/deploy-with-moin.md",
}

// prefixRedirects covers whole old sections.
var prefixRedirects = []struct{ prefix, target string }{
	{"/checkout/", "/"},
	{"/blog/", "/"},
	{"/ops/", "https://moin.run"},
}

type site struct {
	content, changelog, assets string
	dev                        bool
	stars                      atomic.Int64

	mu    sync.Mutex
	pages *markdown
}

// markdown is everything the site reads from Markdown files.
type markdown struct {
	docs      map[string]docsPage
	tree      *docs.DocPage
	privacy   *docs.LegalPage
	changelog string
	llms      string
	sitemap   []byte
}

type docsPage struct {
	page, prev, next *docs.DocPage
}

func newSite(content, changelog, assets string, dev bool) (*site, error) {
	if _, err := os.Stat(filepath.Join(assets, "css", "output.css")); err != nil {
		return nil, errors.New(assets + "/css/output.css is missing: build the CSS with Tailwind first")
	}
	s := &site{content: content, changelog: changelog, assets: assets, dev: dev}
	read, err := s.read()
	if err != nil {
		return nil, err
	}
	s.pages = read
	return s, nil
}

// markdown returns the Markdown pages, read again in dev so an edit shows
// on the next reload.
func (s *site) markdown() (*markdown, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dev {
		read, err := s.read()
		if err != nil {
			return nil, err
		}
		s.pages = read
	}
	return s.pages, nil
}

func (s *site) read() (*markdown, error) {
	result := &markdown{docs: map[string]docsPage{}}

	source, err := os.ReadFile(s.changelog)
	if err != nil {
		return nil, fmt.Errorf("read changelog: %w", err)
	}
	converter := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(html.WithUnsafe()))
	var rendered bytes.Buffer
	if err := converter.Convert(source, &rendered); err != nil {
		return nil, fmt.Errorf("render changelog: %w", err)
	}
	result.changelog = rendered.String()

	service := docs.NewDocsService(s.content, brand.SupportEmail)
	result.tree, err = service.DocsTree()
	if err != nil {
		return nil, fmt.Errorf("read docs: %w", err)
	}
	var llms strings.Builder
	fmt.Fprintf(&llms, "# %s\n\n> %s. One command writes a production Go SaaS onto your machine: login, billing, teams, admin, an API and tests.\n\n## Documentation\n\n", brand.Name, brand.Tagline)
	paths := []string{"/", "/new", "/changelog"}
	for _, page := range service.FlatDocsList() {
		urlPath := docsPath(page)
		prev, next := service.PrevNextPages(page)
		result.docs[urlPath] = docsPage{page: page, prev: prev, next: next}
		paths = append(paths, urlPath)
		description := strings.TrimSpace(page.Description)
		if description == "" {
			description = "goilerplate documentation"
		}
		fmt.Fprintf(&llms, "- [%s](%s%s.md): %s\n", page.Title, brand.URL, urlPath, description)
	}
	fmt.Fprintf(&llms, "\n## Product\n\n- [Stack builder](%s/new): Choose the stack and copy the matching goilerplate new command.\n- [Changelog](%s/changelog): What shipped, newest first.\n- [Source](https://github.com/%s): The CLI, the generator and this website.\n", brand.URL, brand.URL, githubstars.Repo)
	result.llms = llms.String()

	legal := docs.NewLegalService(s.content, brand.SupportEmail)
	if err := legal.LoadPages(); err != nil {
		return nil, fmt.Errorf("read legal pages: %w", err)
	}
	result.privacy, err = legal.Page("privacy")
	if err != nil {
		return nil, fmt.Errorf("read privacy policy: %w", err)
	}
	paths = append(paths, "/legal/privacy")

	result.sitemap, err = sitemap(paths)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// docsPath is the URL of a docs page; the introduction is /docs.
func docsPath(page *docs.DocPage) string {
	if page.Slug == "" || page.Slug == "introduction" {
		return "/docs"
	}
	return "/docs/" + page.Slug
}

func sitemap(paths []string) ([]byte, error) {
	type url struct {
		Loc string `xml:"loc"`
	}
	type urlset struct {
		XMLName xml.Name `xml:"urlset"`
		XMLNS   string   `xml:"xmlns,attr"`
		URLs    []url    `xml:"url"`
	}
	set := urlset{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, p := range paths {
		set.URLs = append(set.URLs, url{Loc: brand.URL + strings.TrimSuffix(p, "/")})
	}
	content, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(content, '\n')...), nil
}

// fetchStars keeps the star count in the navbar fresh. Without GitHub the
// navbar shows none.
func (s *site) fetchStars() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		count, err := githubstars.Fetch(ctx, &http.Client{})
		cancel()
		if err != nil {
			log.Printf("GitHub stars unavailable: %v", err)
		} else {
			s.stars.Store(int64(count))
		}
		time.Sleep(time.Hour)
	}
}

func (s *site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requested := r.URL.Path
	if requested == "/healthz" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.HasPrefix(requested, "/v1/") {
		retiredAPI(w)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clean := path.Clean("/" + requested)
	if clean != "/" && strings.HasSuffix(requested, "/") {
		http.Redirect(w, r, clean, http.StatusMovedPermanently)
		return
	}
	if target, ok := redirects[clean]; ok {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	for _, redirect := range prefixRedirects {
		if strings.HasPrefix(clean, redirect.prefix) {
			http.Redirect(w, r, redirect.target, http.StatusMovedPermanently)
			return
		}
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if strings.HasPrefix(clean, "/assets/") {
		s.asset(w, r, strings.TrimPrefix(clean, "/assets/"))
		return
	}

	md, err := s.markdown()
	if err != nil {
		log.Printf("read Markdown: %v", err)
		http.Error(w, "the site cannot read its pages", http.StatusInternalServerError)
		return
	}
	switch clean {
	case "/":
		s.render(w, r, http.StatusOK, pages.Landing())
		return
	case "/new":
		s.render(w, r, http.StatusOK, pages.Configurator())
		return
	case "/changelog":
		s.render(w, r, http.StatusOK, pages.Changelog(md.changelog))
		return
	case "/legal/privacy":
		s.render(w, r, http.StatusOK, pages.Legal(md.privacy))
		return
	case "/favicon.ico":
		// Browsers, link previews and analytics ask for it at the root.
		s.asset(w, r, "img/favicon.ico")
		return
	case "/robots.txt":
		s.text(w, r, "text/plain; charset=utf-8", []byte("User-agent: *\nAllow: /\nSitemap: "+brand.URL+"/sitemap.xml\n"))
		return
	case "/sitemap.xml":
		s.text(w, r, "application/xml; charset=utf-8", md.sitemap)
		return
	case "/llms.txt":
		s.text(w, r, "text/plain; charset=utf-8", []byte(md.llms))
		return
	}
	if page, ok := md.docs[clean]; ok {
		s.render(w, r, http.StatusOK, pages.Docs(page.page, md.tree, page.prev, page.next))
		return
	}
	if page, ok := md.docs[strings.TrimSuffix(clean, ".md")]; ok && strings.HasSuffix(clean, ".md") {
		s.text(w, r, "text/markdown; charset=utf-8", []byte(page.page.Content))
		return
	}
	s.render(w, r, http.StatusNotFound, pages.NotFound())
}

func (s *site) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	ctx := sitectx.With(r.Context(), sitectx.Page{Path: r.URL.Path, Stars: int(s.stars.Load()), Dev: s.dev})
	var buffer bytes.Buffer
	if err := component.Render(ctx, &buffer); err != nil {
		log.Printf("render %s: %v", r.URL.Path, err)
		http.Error(w, "the page could not be rendered", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(buffer.Bytes())
	}
}

func (s *site) text(w http.ResponseWriter, r *http.Request, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

// asset serves what the pages load. The Tailwind input files and any Go or
// Markdown files stay private; output.css is the stylesheet.
func (s *site) asset(w http.ResponseWriter, r *http.Request, name string) {
	private := strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".go") ||
		(strings.HasPrefix(name, "css/") && name != "css/output.css")
	file := filepath.Join(s.assets, filepath.FromSlash(name))
	info, err := os.Stat(file)
	if private || err != nil || !info.Mode().IsRegular() {
		s.render(w, r, http.StatusNotFound, pages.NotFound())
		return
	}
	switch {
	case s.dev:
		w.Header().Set("Cache-Control", "no-cache")
	case strings.HasPrefix(name, "fonts/"):
		// Fonts load on every page; a long cache keeps text from flashing.
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	default:
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeFile(w, r, file)
}

// retiredAPI answers CLIs from before v4 in the error format they print.
func retiredAPI(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "goilerplate_is_free",
		"message": "goilerplate is free and open source now, no login needed. Install the new CLI: " + brand.Install,
	})
}
