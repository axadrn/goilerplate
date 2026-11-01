package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDatastarRedirectUsesEventStream(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.Header.Set("Datastar-Request", "true")
	recorder := httptest.NewRecorder()

	redirect(recorder, request, "/next", http.StatusSeeOther)

	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `window.location.href = "/next"`) {
		t.Fatalf("redirect event body = %q", body)
	}
}
