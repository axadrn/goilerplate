package api_test

import (
	"bufio"
	"bytes"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/api"
)

var (
	openAPIPathLine   = regexp.MustCompile(`^  (/[^:]*):$`)
	openAPIMethodLine = regexp.MustCompile(`^    (get|post|put|patch|delete):$`)
	openAPITokenRole  = regexp.MustCompile(`^        - apiTokenAuth: \[(read|write)\]$`)
	pathParameter     = regexp.MustCompile(`\{[^}]+\}`)
)

// documentedOperations returns every "METHOD /path" of the OpenAPI document
// under /api/v1.
func documentedOperations(t *testing.T) []string {
	t.Helper()
	var operations []string
	currentPath := ""
	scanner := bufio.NewScanner(bytes.NewReader(api.OpenAPIDocument))
	for scanner.Scan() {
		line := scanner.Text()
		pathMatch := openAPIPathLine.FindStringSubmatch(line)
		if pathMatch != nil {
			currentPath = pathMatch[1]
			continue
		}
		if line != "" && !strings.HasPrefix(line, "    ") {
			currentPath = ""
			continue
		}
		methodMatch := openAPIMethodLine.FindStringSubmatch(line)
		if methodMatch != nil && strings.HasPrefix(currentPath, "/api/v1/") {
			operations = append(operations, strings.ToUpper(methodMatch[1])+" "+currentPath)
		}
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) == 0 {
		t.Fatal("the OpenAPI document lists no operations under /api/v1")
	}
	return operations
}

// TestOpenAPIDocumentsEveryRoute keeps the router and the OpenAPI document
// in sync: every route has an operation and every operation has a route.
func TestOpenAPIDocumentsEveryRoute(t *testing.T) {
	environment := newAPIEnvironment(t)
	documented := documentedOperations(t)

	registered := make([]string, 0, len(environment.routes))
	for _, route := range environment.routes {
		if slices.Contains(registered, route.Pattern) {
			t.Errorf("route %q is registered twice", route.Pattern)
		}
		registered = append(registered, route.Pattern)
		if !strings.Contains(route.Pattern, " /api/v1/") {
			t.Errorf("route %q is not under /api/v1", route.Pattern)
		}
	}

	for _, pattern := range registered {
		if !slices.Contains(documented, pattern) {
			t.Errorf("route %q is missing from openapi.yaml", pattern)
		}
	}
	for _, operation := range documented {
		if !slices.Contains(registered, operation) {
			t.Errorf("openapi.yaml documents %q, but the router has no such route", operation)
		}
	}
}

// TestOpenAPIOperationsReachTheirRoute sends every documented operation
// through the router and checks that it reaches a route instead of the API
// not found handler.
func TestOpenAPIOperationsReachTheirRoute(t *testing.T) {
	environment := newAPIEnvironment(t)
	for _, operation := range documentedOperations(t) {
		method, path, _ := strings.Cut(operation, " ")
		path = pathParameter.ReplaceAllString(path, "unknown")
		recorder := environment.send(t, apiRequest{method: method, path: path})
		if recorder.Code == http.StatusMethodNotAllowed || strings.Contains(recorder.Body.String(), "The requested API endpoint does not exist.") {
			t.Errorf("%s did not reach a route: %d %s", operation, recorder.Code, recorder.Body.String())
		}
	}
}

func TestOpenAPIDocumentIsServed(t *testing.T) {
	environment := newAPIEnvironment(t)

	recorder := environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/openapi.yaml"})

	expectStatus(t, recorder, http.StatusOK)
	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/yaml" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("openapi: 3.1.0\n")) {
		t.Fatalf("body does not start with the OpenAPI version: %.40q", recorder.Body.String())
	}
}

// documentedTokenAccess describes what the OpenAPI document says about
// personal API tokens on one operation.
type documentedTokenAccess struct {
	// role is the scope in its apiTokenAuth entry, empty when it has none.
	role string
	// public is set for operations with "security: []".
	public bool
	// forbidden is set when the operation documents a 403 answer.
	forbidden bool
}

// documentedTokenAccesses reads the apiTokenAuth role, the public marker and
// the 403 answer of every operation under /api/v1.
func documentedTokenAccesses(t *testing.T) map[string]documentedTokenAccess {
	t.Helper()
	accesses := make(map[string]documentedTokenAccess)
	currentPath := ""
	currentOperation := ""
	scanner := bufio.NewScanner(bytes.NewReader(api.OpenAPIDocument))
	for scanner.Scan() {
		line := scanner.Text()
		pathMatch := openAPIPathLine.FindStringSubmatch(line)
		if pathMatch != nil {
			currentPath = pathMatch[1]
			currentOperation = ""
			continue
		}
		if line != "" && !strings.HasPrefix(line, "    ") {
			currentPath = ""
			currentOperation = ""
			continue
		}
		methodMatch := openAPIMethodLine.FindStringSubmatch(line)
		if methodMatch != nil && strings.HasPrefix(currentPath, "/api/v1/") {
			currentOperation = strings.ToUpper(methodMatch[1]) + " " + currentPath
			accesses[currentOperation] = documentedTokenAccess{}
			continue
		}
		if currentOperation == "" {
			continue
		}
		access := accesses[currentOperation]
		roleMatch := openAPITokenRole.FindStringSubmatch(line)
		switch {
		case roleMatch != nil:
			access.role = roleMatch[1]
		case line == "      security: []":
			access.public = true
		case line == "        '403':":
			access.forbidden = true
		}
		accesses[currentOperation] = access
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
	return accesses
}

// TestOpenAPIDocumentsTokenAccess keeps the apiTokenAuth entries of the
// document in sync with the token gate of every route: an operation that
// needs a session lists no apiTokenAuth, and every operation that can refuse
// a token documents the 403.
func TestOpenAPIDocumentsTokenAccess(t *testing.T) {
	environment := newAPIEnvironment(t)
	documented := documentedTokenAccesses(t)
	roles := map[api.TokenAccess]string{api.TokenDenied: "", api.TokenRead: "read", api.TokenWrite: "write"}
	for _, route := range environment.routes {
		access, ok := documented[route.Pattern]
		if !ok {
			continue
		}
		if !access.public && access.role != roles[route.Tokens] {
			t.Errorf("%s: openapi.yaml apiTokenAuth role %q, route allows %q", route.Pattern, access.role, roles[route.Tokens])
		}
		if access.public && access.role != "" {
			t.Errorf("%s: a public operation lists apiTokenAuth", route.Pattern)
		}
		if route.Tokens != api.TokenRead && !access.forbidden {
			t.Errorf("%s: can answer 403 to tokens, but openapi.yaml documents no 403", route.Pattern)
		}
	}
}

// TestTokenScopesFollowTheMethod checks that only GET routes are open to
// read tokens and every route that changes data needs the write scope.
func TestTokenScopesFollowTheMethod(t *testing.T) {
	environment := newAPIEnvironment(t)
	for _, route := range environment.routes {
		method, _, _ := strings.Cut(route.Pattern, " ")
		if method == http.MethodGet && route.Tokens == api.TokenWrite {
			t.Errorf("%s needs the write scope for a read", route.Pattern)
		}
		if method != http.MethodGet && route.Tokens == api.TokenRead {
			t.Errorf("%s changes data with the read scope", route.Pattern)
		}
	}
}
