package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMXRedirectUsesResponseHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()

	redirect(recorder, request, "/next", http.StatusSeeOther)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if location := recorder.Header().Get("HX-Redirect"); location != "/next" {
		t.Fatalf("HX-Redirect = %q, want /next", location)
	}
}
