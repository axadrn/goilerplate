package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const (
	testInlineScript = "\n\t\t\t{\n\t\t\t\t__sveltekit_x = { base: \"\" };\n\t\t\t\tPromise.all([import(\"/_app/immutable/entry/start.abc.js\")]);\n\t\t\t}\n\t\t"
	testIndex        = `<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<script src="/theme.js"></script>
		<link rel="modulepreload" href="/_app/immutable/entry/start.abc.js">
	</head>
	<body>
		<div style="display: contents">
			<script>` + testInlineScript + `</script>
		</div>
	</body>
</html>
`
)

func testBuild() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                        {Data: []byte(testIndex)},
		"theme.js":                          {Data: []byte("document.documentElement.dataset.theme = 'light';\n")},
		"favicon.svg":                       {Data: []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")},
		"_app/version.json":                 {Data: []byte(`{"version":"1"}`)},
		"_app/immutable/entry/start.abc.js": {Data: []byte("export const start = 1;\n")},
		"_app/immutable/assets/app.def.css": {Data: []byte("body{margin:0}\n")},
		".gitkeep":                          {Data: []byte{}},
	}
}

func testScriptHash(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

func serveSPA(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

func TestSPAServesIndexForPages(t *testing.T) {
	t.Parallel()
	handler := NewSPA(testBuild(), SPAConfig{})

	for _, target := range []string{"/", "/app", "/app/settings", "/app/acme/settings/", "/auth/magic-link/0123abcd", "/nope", "/index.html", "/_app/immutable"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			recorder := serveSPA(handler, http.MethodGet, target)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			if recorder.Body.String() != testIndex {
				t.Fatalf("body = %q, want index.html", recorder.Body.String())
			}
			expectHeader(t, recorder, "Cache-Control", "no-cache")
			expectHeader(t, recorder, "Content-Type", "text/html; charset=utf-8")
		})
	}
}

func TestSPAAnswersHeadWithoutBody(t *testing.T) {
	t.Parallel()
	recorder := serveSPA(NewSPA(testBuild(), SPAConfig{}), http.MethodHead, "/app/settings")

	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
	expectHeader(t, recorder, "Cache-Control", "no-cache")
}

func TestSPARejectsOtherMethods(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		recorder := serveSPA(NewSPA(testBuild(), SPAConfig{}), method, "/app/settings")

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", method, recorder.Code)
		}
		expectHeader(t, recorder, "Allow", "GET, HEAD")
		if strings.Contains(recorder.Body.String(), "<html") {
			t.Fatalf("%s served index.html", method)
		}
	}
}

func TestSPAServesFiles(t *testing.T) {
	t.Parallel()
	handler := NewSPA(testBuild(), SPAConfig{})
	tests := []struct {
		target      string
		body        string
		contentType string
		cache       string
	}{
		{target: "/_app/immutable/entry/start.abc.js", body: "export const start = 1;\n", contentType: "text/javascript; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{target: "/_app/immutable/assets/app.def.css", body: "body{margin:0}\n", contentType: "text/css; charset=utf-8", cache: "public, max-age=31536000, immutable"},
		{target: "/_app/version.json", body: `{"version":"1"}`, contentType: "application/json"},
		{target: "/theme.js", body: "document.documentElement.dataset.theme = 'light';\n", contentType: "text/javascript; charset=utf-8"},
		{target: "/favicon.svg", body: "<svg xmlns=\"http://www.w3.org/2000/svg\"/>", contentType: "image/svg+xml"},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			t.Parallel()
			recorder := serveSPA(handler, http.MethodGet, test.target)

			if recorder.Code != http.StatusOK || recorder.Body.String() != test.body {
				t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
			}
			expectHeader(t, recorder, "Content-Type", test.contentType)
			expectHeader(t, recorder, "Cache-Control", test.cache)
		})
	}
}

func TestSPAAnswersMissingFilesWith404(t *testing.T) {
	t.Parallel()
	handler := NewSPA(testBuild(), SPAConfig{})

	for _, target := range []string{"/nope.js", "/app/nope.css", "/_app/immutable/entry/gone.js", "/.gitkeep", "/_app/immutable/.hidden.js", "/_app.d/"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			recorder := serveSPA(handler, http.MethodGet, target)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", recorder.Code)
			}
			if strings.Contains(recorder.Body.String(), "<html") {
				t.Fatal("a missing file got index.html")
			}
		})
	}
}

// TestSPANeverAnswersServerPaths keeps the fallback away from the paths of
// the Go server, even for methods and subpaths that have no route.
func TestSPANeverAnswersServerPaths(t *testing.T) {
	t.Parallel()
	handler := NewSPA(testBuild(), SPAConfig{})

	for _, target := range []string{"/api", "/api/", "/api/v1/nope", "/mcp", "/mcp/x", "/webhooks/payment", "/healthz", "/auth/google", "/auth/google/callback", "/auth/github", "/auth/github/callback"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(method+" "+target, func(t *testing.T) {
				t.Parallel()
				recorder := serveSPA(handler, method, target)

				if recorder.Code != http.StatusNotFound {
					t.Fatalf("status = %d, want 404", recorder.Code)
				}
				contentType := recorder.Header().Get("Content-Type")
				if !strings.HasPrefix(contentType, "application/json") {
					t.Fatalf("Content-Type = %q, want the JSON envelope", contentType)
				}
				if strings.Contains(recorder.Body.String(), "<html") {
					t.Fatal("a server path got index.html")
				}
			})
		}
	}
}

func TestSPAPolicyHashesInlineScripts(t *testing.T) {
	t.Parallel()
	recorder := serveSPA(NewSPA(testBuild(), SPAConfig{}), http.MethodGet, "/app")

	want := "default-src 'self'; script-src 'self' " + testScriptHash(testInlineScript) + "; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	expectHeader(t, recorder, "Content-Security-Policy", want)
	file := serveSPA(NewSPA(testBuild(), SPAConfig{}), http.MethodGet, "/_app/immutable/entry/start.abc.js")
	expectHeader(t, file, "Content-Security-Policy", want)
}

func TestInlineScriptHashes(t *testing.T) {
	t.Parallel()
	document := []byte(`<script src="/theme.js"></script><SCRIPT type="module">one()</SCRIPT><script>two()</script><script>one()</script><script nonce="x" data-src="y">three()</script>`)

	got := inlineScriptHashes(document)

	want := []string{testScriptHash("one()"), testScriptHash("two()"), testScriptHash("three()")}
	if len(got) != len(want) {
		t.Fatalf("hashes = %v, want %d", got, len(want))
	}
	for index := range want {
		if "'sha256-"+got[index]+"'" != want[index] {
			t.Fatalf("hash %d = %q, want %s", index, got[index], want[index])
		}
	}
}

func TestSPAPolicyAllowsStorageImages(t *testing.T) {
	t.Parallel()
	recorder := serveSPA(NewSPA(testBuild(), SPAConfig{S3Endpoint: "http://localhost:9000"}), http.MethodGet, "/app/settings")

	policy := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "; img-src 'self' data: https: http://localhost:9000; ") {
		t.Fatalf("Content-Security-Policy = %q", policy)
	}
}

func TestSPAWithoutBuildAnswers503(t *testing.T) {
	t.Parallel()
	build := fstest.MapFS{
		".gitkeep":         {Data: []byte{}},
		"_app/version.txt": {Data: []byte("1")},
	}
	handler := NewSPA(build, SPAConfig{})

	page := serveSPA(handler, http.MethodGet, "/app/settings")
	if page.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", page.Code)
	}
	if !strings.Contains(page.Body.String(), "Run task build or task dev.") {
		t.Fatalf("body = %q", page.Body.String())
	}
	contentType := page.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	policy := page.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "script-src 'self';") {
		t.Fatalf("Content-Security-Policy = %q, want no hashes", policy)
	}
	missing := serveSPA(handler, http.MethodGet, "/nope.js")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing file status = %d, want 404", missing.Code)
	}
	file := serveSPA(handler, http.MethodGet, "/_app/version.txt")
	if file.Code != http.StatusOK {
		t.Fatalf("file status = %d, want 200", file.Code)
	}
	unknownAPI := serveSPA(handler, http.MethodGet, "/api/v1/nope")
	if unknownAPI.Code != http.StatusNotFound {
		t.Fatalf("api status = %d, want 404", unknownAPI.Code)
	}
}

func expectHeader(t *testing.T, recorder *httptest.ResponseRecorder, name, want string) {
	t.Helper()
	got := recorder.Header().Get(name)
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}
