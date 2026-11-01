package routes

import (
	"os"
	"strings"
	"testing"
)

// TestAPIRoutesComeFromTable keeps every /api/v1 route in api.Routes, where
// internal/api/openapi_test.go checks it against the OpenAPI document. A
// route registered here directly would escape that check.
func TestAPIRoutesComeFromTable(t *testing.T) {
	source, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "/api/v1") {
		t.Fatal("routes.go registers an /api/v1 route directly; add it to api.Routes and openapi.yaml instead")
	}
	if !strings.Contains(string(source), "api.Routes(") {
		t.Fatal("routes.go does not register api.Routes")
	}
}
