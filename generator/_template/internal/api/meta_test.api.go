package api_test

import (
	"net/http"
	"testing"

	"{{.ModulePath}}/internal/brand"
)

func TestMetaNamesTheProjectAndTheSupportAddress(t *testing.T) {
	environment := newAPIEnvironment(t)

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/meta"})

	expectStatus(t, recorder, http.StatusOK)
	body := decodeBody[map[string]string](t, recorder)
	if len(body) != 2 || body["name"] != brand.Name || body["support_email"] != testSupportEmail {
		t.Fatalf("body = %v", body)
	}
}

func TestMetaIsOpenToReadTokens(t *testing.T) {
	environment := newAPIEnvironment(t)
	owner := seedTokenOwner(t, environment, "reader")
	created := createAPIToken(t, environment, owner, `"name":"Reader"`)

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/meta", bearer: created.Token})

	expectStatus(t, recorder, http.StatusOK)
}
