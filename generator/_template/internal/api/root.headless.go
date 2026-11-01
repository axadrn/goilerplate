package api

import (
	"net/http"

	"{{.ModulePath}}/internal/brand"
)

type rootResponse struct {
	Name string `json:"name"`
	API  string `json:"api"`
}

// Root answers GET / of a headless project with the project name and the
// base path of the JSON API.
func Root(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, rootResponse{Name: brand.Name, API: "/api/v1"})
}
