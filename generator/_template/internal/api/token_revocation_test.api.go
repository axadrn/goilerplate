package api_test

import (
	"net/http"
	"testing"
)

func TestForgotPasswordRevokesAPITokens(t *testing.T) {
	environment := newAPIEnvironment(t)
	owner := seedTokenOwner(t, environment, "recovering")
	created := createAPIToken(t, environment, owner, `"name":"Script"`)

	expectStatus(t, environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/auth/forgot-password", body: `{"email":"recovering@example.com"}`}), http.StatusAccepted)
	token := environment.mailedToken(t, "/auth/forgot-password/")
	expectStatus(t, environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/auth/forgot-password/verify", body: `{"token":"` + token + `"}`}), http.StatusOK)

	expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: created.Token}), http.StatusUnauthorized, "unauthenticated")
}

func TestPasswordChangesRevokeAPITokens(t *testing.T) {
	environment := newAPIEnvironment(t)
	owner := seedTokenOwner(t, environment, "changer")

	for _, change := range []apiRequest{
		{method: http.MethodPost, path: "/api/v1/account/password", body: `{"current_password":"` + testPassword + `","new_password":"another long passphrase"}`},
		{method: http.MethodDelete, path: "/api/v1/account/password"},
		{method: http.MethodPost, path: "/api/v1/account/password/set", body: `{"password":"` + testPassword + `"}`},
	} {
		created := createAPIToken(t, environment, owner, `"name":"Script"`)
		expectStatus(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: created.Token}), http.StatusOK)

		change.bearer = owner.session
		expectStatus(t, environment.send(t, change), http.StatusNoContent)

		expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: created.Token}), http.StatusUnauthorized, "unauthenticated")
		expectStatus(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: owner.session}), http.StatusOK)
	}
}

func TestSignOutEverywhereKeepsAPITokens(t *testing.T) {
	environment := newAPIEnvironment(t)
	owner := seedTokenOwner(t, environment, "keeper")
	created := createAPIToken(t, environment, owner, `"name":"Script"`)
	other := environment.signIn(t, owner.id)

	expectStatus(t, environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/sessions", bearer: owner.session}), http.StatusNoContent)

	expectError(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: other}), http.StatusUnauthorized, "unauthenticated")
	expectStatus(t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: created.Token}), http.StatusOK)
}
