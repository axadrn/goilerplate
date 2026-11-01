package handler

import (
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/pages"
)

type BlogHandler struct {
	blogService *service.BlogService
}

func NewBlogHandler(blogService *service.BlogService) *BlogHandler {
	return &BlogHandler{
		blogService: blogService,
	}
}

func (h *BlogHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.blogService.Posts()
	if err != nil {
		slog.Error("failed to load blog posts", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Blog unavailable", "Blog posts could not be loaded. Please try again."))
		return
	}

	render(w, r, pages.BlogList(posts))
}

func (h *BlogHandler) ShowPost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, pages.NotFound())
		return
	}

	post, err := h.blogService.Post(slug)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, pages.NotFound())
		return
	}

	render(w, r, pages.BlogPost(post))
}

func (h *BlogHandler) ListByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	if tag == "" {
		w.WriteHeader(http.StatusNotFound)
		render(w, r, pages.NotFound())
		return
	}

	posts, err := h.blogService.PostsByTag(tag)
	if err != nil {
		slog.Error("failed to load blog posts", "error", err, "tag", tag)
		w.WriteHeader(http.StatusInternalServerError)
		render(w, r, pages.ErrorPage(http.StatusInternalServerError, "Blog unavailable", "Blog posts could not be loaded. Please try again."))
		return
	}

	render(w, r, pages.BlogList(posts, tag))
}
