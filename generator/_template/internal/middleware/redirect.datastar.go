package middleware

import (
	"log/slog"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

func redirect(w http.ResponseWriter, r *http.Request, location string, status int) {
	if r.Header.Get("Datastar-Request") != "true" {
		http.Redirect(w, r, location, status)
		return
	}
	if err := datastar.NewSSE(w, r).Redirect(location); err != nil {
		slog.Error("render Datastar redirect failed", "error", err)
	}
}
