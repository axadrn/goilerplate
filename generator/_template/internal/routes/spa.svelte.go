package routes

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"{{.ModulePath}}/internal/middleware"
)

// SPAConfig holds the runtime values the SvelteKit app policy needs.
type SPAConfig struct {
	// S3Endpoint is added to img-src so avatars from storage load.
	S3Endpoint string
}

// spaReservedPaths belong to the Go server. The app never answers them, even
// for a method or a subpath the server has no route for.
var spaReservedPaths = []string{"/api", "/mcp", "/webhooks", "/healthz", "/auth/google", "/auth/github"}

var (
	inlineScriptPattern = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script`)
	scriptSourcePattern = regexp.MustCompile(`(?i)(?:^|\s)src\s*=`)
)

// spaHandler serves the static SvelteKit build. Paths whose last segment has
// no dot are app pages and get index.html, every other path is a file.
type spaHandler struct {
	files  fs.FS
	index  []byte
	policy string
}

// NewSPA serves the SvelteKit build in files. Without index.html, pages
// answer 503 until the app is built, and files are still served.
func NewSPA(files fs.FS, cfg SPAConfig) http.Handler {
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		slog.Warn("web/build has no index.html, pages answer 503. Run task build or task dev.", "error", err)
		index = nil
	}
	return &spaHandler{files: files, index: index, policy: spaPolicy(index, cfg)}
}

// spaPolicy allows the app's own files, the inline scripts of index.html by
// their hash, and nothing from other origins except images.
func spaPolicy(index []byte, cfg SPAConfig) string {
	scriptSources := []string{"script-src 'self'"}
	for _, hash := range inlineScriptHashes(index) {
		scriptSources = append(scriptSources, "'sha256-"+hash+"'")
	}
	imageSources := "img-src 'self' data: https:"
	if cfg.S3Endpoint != "" {
		imageSources += " " + cfg.S3Endpoint
	}
	return strings.Join([]string{
		"default-src 'self'",
		strings.Join(scriptSources, " "),
		"style-src 'self' 'unsafe-inline'",
		imageSources,
		"font-src 'self'",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

// inlineScriptHashes returns the base64 SHA-256 of every inline script in
// an HTML document, in document order and without duplicates.
func inlineScriptHashes(document []byte) []string {
	var hashes []string
	for _, match := range inlineScriptPattern.FindAllSubmatch(document, -1) {
		if scriptSourcePattern.Match(match[1]) {
			continue
		}
		sum := sha256.Sum256(match[2])
		hash := base64.StdEncoding.EncodeToString(sum[:])
		if !slices.Contains(hashes, hash) {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleaned := path.Clean("/" + r.URL.Path)
	if isSPAReservedPath(cleaned) {
		middleware.APINotFound(w, r)
		return
	}
	w.Header().Set("Content-Security-Policy", h.policy)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(cleaned, "/")
	if !strings.Contains(path.Base(cleaned), ".") {
		h.serveIndex(w, r)
		return
	}
	if name == "index.html" {
		h.serveIndex(w, r)
		return
	}
	h.serveFile(w, r, name)
}

func isSPAReservedPath(cleaned string) bool {
	for _, reserved := range spaReservedPaths {
		if cleaned == reserved || strings.HasPrefix(cleaned, reserved+"/") {
			return true
		}
	}
	return false
}

func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "The web app is not built. Run task build or task dev.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(h.index))
}

func (h *spaHandler) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	for _, element := range strings.Split(name, "/") {
		if strings.HasPrefix(element, ".") {
			http.NotFound(w, r)
			return
		}
	}
	file, err := h.files.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	content, seekable := file.(io.ReadSeeker)
	if !seekable {
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		content = bytes.NewReader(data)
	}
	if strings.HasPrefix(name, "_app/immutable/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, name, time.Time{}, content)
}
