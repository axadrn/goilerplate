package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"{{.ModulePath}}/internal/api"
	"{{.ModulePath}}/internal/brand"
)

func TestRootNamesTheProjectAndTheAPI(t *testing.T) {
	recorder := httptest.NewRecorder()

	api.Root(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body map[string]string
	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	if len(body) != 2 || body["name"] != brand.Name || body["api"] != "/api/v1" {
		t.Fatalf("body = %v", body)
	}
}
