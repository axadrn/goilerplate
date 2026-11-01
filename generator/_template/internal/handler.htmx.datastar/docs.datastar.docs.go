package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/components/toast"
	"{{.ModulePath}}/internal/ui/pages"
)

type DocsHandler struct {
	docsService *service.DocsService
}

func NewDocsHandler(docsService *service.DocsService) *DocsHandler {
	handler := &DocsHandler{
		docsService: docsService,
	}

	// Build the docs tree on initialization
	err := handler.docsService.BuildDocsTree()
	if err != nil {
		// Silently continue. Docs might be added later
		_ = err
	}

	return handler
}

func (h *DocsHandler) ShowDocs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	slug := ""
	markdown := strings.HasSuffix(path, ".md")
	if markdown {
		path = strings.TrimSuffix(path, ".md")
	} else {
		w.Header().Add("Vary", "Datastar-Request")
	}

	if path != "/docs" && path != "/docs/" {
		slug = strings.TrimPrefix(path, "/docs/")
		slug = strings.TrimSuffix(slug, "/")
	}
	if slug == "introduction" {
		target := "/docs"
		if markdown {
			target = "/docs.md"
		}
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
		return
	}
	// Get the requested page
	page, err := h.docsService.DocPage(slug)
	if err != nil {
		if markdown {
			http.Error(w, "Documentation page not found", http.StatusNotFound)
			return
		}
		if isAction(r) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		render(w, r, pages.NotFound())
		return
	}
	if markdown {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = w.Write([]byte(page.Content))
		return
	}

	// Get the full docs tree for navigation
	docsTree, err := h.docsService.DocsTree()
	if err != nil {
		if isAction(r) {
			appendElement(w, r, toast.Toast(toast.Props{Title: "Documentation unavailable", Description: "The documentation could not be loaded. Please try again.", Type: toast.TypeError}), "#toast-container")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Documentation unavailable", "The documentation could not be loaded. Please try again."))
		}
		return
	}

	// Get previous and next pages
	prevPage, nextPage := h.docsService.PrevNextPages(page)

	if isAction(r) {
		if err := renderFragment(w, r, pages.Docs(page, docsTree, prevPage, nextPage), "docs-content", "seo-title", "seo-markdown", "docs-breadcrumb", "docs-toc"); err == nil {
			if err := datastar.NewSSE(w, r).DispatchCustomEvent("goilerplate:docs:navigated", map[string]string{"path": r.URL.Path}); err != nil {
				slog.Error("dispatch docs navigation event failed", "error", err)
			}
		}
	} else {
		render(w, r, pages.Docs(page, docsTree, prevPage, nextPage))
	}
}
