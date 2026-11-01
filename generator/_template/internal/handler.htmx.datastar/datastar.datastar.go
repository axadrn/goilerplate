package handler

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

func isAction(r *http.Request) bool {
	return r.Header.Get("Datastar-Request") == "true"
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if !isAction(r) {
		renderComponent(w, r, component)
		return
	}
	if err := datastar.NewSSE(w, r).PatchElementTempl(component); err != nil {
		slog.Error("render Datastar elements failed", "error", err)
	}
}

func renderFragment(w http.ResponseWriter, r *http.Request, component templ.Component, fragmentIDs ...any) error {
	if !isAction(r) {
		if err := templ.RenderFragments(r.Context(), w, component, fragmentIDs...); err != nil {
			renderError(w, "render fragment failed", err)
			return err
		}
		return nil
	}

	var rendered bytes.Buffer
	if err := templ.RenderFragments(r.Context(), &rendered, component, fragmentIDs...); err != nil {
		renderError(w, "render fragment failed", err)
		return err
	}
	if err := datastar.NewSSE(w, r).PatchElements(rendered.String()); err != nil {
		slog.Error("render Datastar fragments failed", "error", err)
		return err
	}
	return nil
}

func appendElement(w http.ResponseWriter, r *http.Request, component templ.Component, selector string) {
	if !isAction(r) {
		renderComponent(w, r, component)
		return
	}
	if err := datastar.NewSSE(w, r).PatchElementTempl(component, datastar.WithSelector(selector), datastar.WithModeAppend()); err != nil {
		slog.Error("render Datastar append failed", "error", err)
	}
}

func redirect(w http.ResponseWriter, r *http.Request, location string, status int) {
	if !isAction(r) {
		http.Redirect(w, r, location, status)
		return
	}
	if err := datastar.NewSSE(w, r).Redirect(location); err != nil {
		slog.Error("render Datastar redirect failed", "error", err)
	}
}

func renderComponent(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if err := component.Render(r.Context(), w); err != nil {
		renderError(w, "render failed", err)
	}
}

func renderError(w http.ResponseWriter, message string, err error) {
	slog.Error(message, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
