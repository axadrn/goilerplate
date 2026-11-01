package handler

import (
	"net/http"

	"{{.ModulePath}}/internal/ui/pages"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.Dashboard())
}
