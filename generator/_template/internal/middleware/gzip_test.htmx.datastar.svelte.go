package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestGzip(t *testing.T) {
	t.Parallel()

	script := strings.Repeat("console.log('goilerplate');\n", 200)
	files := http.FileServer(http.FS(fstest.MapFS{
		"app.js":     {Data: []byte(script)},
		"font.woff2": {Data: []byte("wOF2 font bytes")},
	}))
	handler := Gzip(files)

	tests := []struct {
		name           string
		path           string
		acceptEncoding string
		rangeHeader    string
		wantStatus     int
		wantEncoding   string
		wantVary       string
	}{
		{name: "script with gzip", path: "/app.js", acceptEncoding: "gzip, deflate, br", wantStatus: http.StatusOK, wantEncoding: "gzip", wantVary: "Accept-Encoding"},
		{name: "script without gzip", path: "/app.js", wantStatus: http.StatusOK, wantVary: "Accept-Encoding"},
		{name: "script with gzip refused", path: "/app.js", acceptEncoding: "gzip;q=0, br", wantStatus: http.StatusOK, wantVary: "Accept-Encoding"},
		{name: "script range serves whole file", path: "/app.js", acceptEncoding: "gzip", rangeHeader: "bytes=0-9", wantStatus: http.StatusOK, wantEncoding: "gzip", wantVary: "Accept-Encoding"},
		{name: "font stays as is", path: "/font.woff2", acceptEncoding: "gzip", wantStatus: http.StatusOK},
		{name: "missing script", path: "/missing.js", acceptEncoding: "gzip", wantStatus: http.StatusNotFound, wantVary: "Accept-Encoding"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", test.acceptEncoding)
			}
			if test.rangeHeader != "" {
				request.Header.Set("Range", test.rangeHeader)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if got := response.Header().Get("Content-Encoding"); got != test.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, test.wantEncoding)
			}
			if got := response.Header().Get("Vary"); got != test.wantVary {
				t.Fatalf("Vary = %q, want %q", got, test.wantVary)
			}
			if test.wantEncoding != "gzip" {
				return
			}
			if got := response.Header().Get("Content-Length"); got != "" {
				t.Fatalf("Content-Length = %q, want none", got)
			}
			reader, err := gzip.NewReader(response.Body)
			if err != nil {
				t.Fatalf("gzip.NewReader: %v", err)
			}
			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("read gzip body: %v", err)
			}
			if string(body) != script {
				t.Fatalf("body has %d bytes, want the %d byte script", len(body), len(script))
			}
		})
	}
}
