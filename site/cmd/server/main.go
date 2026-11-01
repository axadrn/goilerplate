// Command server serves the built website: the files in a directory, the
// redirects of old URLs, a 404 page and a plain answer for CLIs from before
// v4, which still call the retired API.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/axadrn/goilerplate/v4/site/brand"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	dir := flag.String("dir", "dist", "directory with the built site")
	addr := flag.String("addr", ":8090", "address to listen on")
	flag.Parse()

	server := &http.Server{
		Addr:              *addr,
		Handler:           newSite(*dir),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("serving %s on %s", *dir, *addr)
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
	"/pricing":                     "/",
	"/license":                     "/",
	"/affiliates":                  "/",
	"/legal/terms":                 "/legal/privacy",
	"/ops":                         "https://ops.goilerplate.com",
	"/blog":                        "/",
	"/docs/introduction":           "/docs",
	"/docs/introduction.md":        "/docs.md",
	"/docs/guides/cli-accounts":    "/docs/installation",
	"/docs/guides/cli-accounts.md": "/docs/installation.md",
}

// prefixRedirects covers whole old sections.
var prefixRedirects = []struct{ prefix, target string }{
	{"/checkout/", "/"},
	{"/blog/", "/"},
	{"/ops/", "https://ops.goilerplate.com"},
}

type site struct {
	dir string
}

func newSite(dir string) http.Handler {
	return &site{dir: dir}
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

	name, ok := s.resolve(clean)
	if !ok {
		s.notFound(w, r)
		return
	}
	if strings.HasPrefix(clean, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	if strings.HasSuffix(name, ".md") {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, name)
}

// resolve finds the file for a clean path: the file itself, or the page
// rendered for it, such as docs/installation.html for /docs/installation.
func (s *site) resolve(clean string) (string, bool) {
	if clean == "/" {
		clean = "/index"
	}
	base := filepath.Join(s.dir, filepath.FromSlash(clean))
	for _, candidate := range []string{base, base + ".html"} {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() {
			return candidate, true
		}
	}
	return "", false
}

func (s *site) notFound(w http.ResponseWriter, r *http.Request) {
	page, err := os.ReadFile(filepath.Join(s.dir, "404.html"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if r.Method != http.MethodHead {
		_, _ = w.Write(page)
	}
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
