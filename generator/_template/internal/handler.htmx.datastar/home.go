package handler

import (
	"net/http"

	"{{.ModulePath}}/internal/ui/pages"
)

type HomeHandler struct{}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

func (h *HomeHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.Home())
}

func (h *HomeHandler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	render(w, r, pages.NotFound())
}
