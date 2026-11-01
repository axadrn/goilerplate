package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/ui/blocks"
	"{{.ModulePath}}/internal/validation"
)

type newsletterHandler struct {
	newsletterService *service.NewsletterService
}

func NewNewsletterHandler(newsletterService *service.NewsletterService) *newsletterHandler {
	return &newsletterHandler{
		newsletterService: newsletterService,
	}
}

func (h *newsletterHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	err := validation.ValidateEmail(email)
	if err != nil {
		render(w, r, blocks.FooterNewsletterForm("Please provide a valid email address"))
		return
	}

	err = h.newsletterService.Subscribe(email)
	if err != nil {
		slog.Warn("newsletter subscription error", "error", err)
	}

	// Always show success (prevents email enumeration)
	render(w, r, blocks.FooterNewsletterSuccess())
}
