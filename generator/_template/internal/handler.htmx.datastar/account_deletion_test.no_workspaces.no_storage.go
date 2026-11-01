package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/service"
)

func TestAccountDeletionErrorShowsActiveSubscriptionWithoutStorage(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest("DELETE", "/app/account", nil)
	accountDeletionError(response, request, "user-1", service.ErrActiveSubscription)
	if !strings.Contains(response.Body.String(), "Active Subscription") {
		t.Fatalf("response = %q", response.Body.String())
	}
}
