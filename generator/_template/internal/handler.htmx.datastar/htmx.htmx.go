package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

func isAction(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if err := component.Render(r.Context(), w); err != nil {
		renderError(w, "render failed", err)
	}
}

func renderFragment(w http.ResponseWriter, r *http.Request, component templ.Component, fragmentIDs ...any) {
	if err := templ.RenderFragments(r.Context(), w, component, fragmentIDs...); err != nil {
		renderError(w, "render fragment failed", err)
	}
}

func renderOOB(w http.ResponseWriter, r *http.Request, component templ.Component, target string) {
	if _, err := fmt.Fprintf(w, `<div hx-swap-oob="%s">`, target); err != nil {
		slog.Error("render oob wrapper failed", "error", err)
		return
	}
	if err := component.Render(r.Context(), w); err != nil {
		renderError(w, "render oob failed", err)
		return
	}
	if _, err := w.Write([]byte(`</div>`)); err != nil {
		slog.Error("render oob wrapper failed", "error", err)
	}
}

func redirect(w http.ResponseWriter, r *http.Request, location string, status int) {
	if isAction(r) {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, location, status)
}

func renderError(w http.ResponseWriter, message string, err error) {
	slog.Error(message, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
