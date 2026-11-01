package api

import (
	_ "embed"
	"log/slog"
	"net/http"
)

// OpenAPIDocument is the OpenAPI 3.1 description of this API. It is
// rendered for the modules of this project and kept in sync with Routes by
// openapi_test.go.
//
//go:embed openapi.yaml
var OpenAPIDocument []byte

// OpenAPI serves OpenAPIDocument.
func OpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "no-cache")
	_, err := w.Write(OpenAPIDocument)
	if err != nil {
		slog.Debug("write openapi document failed", "error", err)
	}
}
