package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"{{.ModulePath}}/internal/model"
)

func TestSidebarIdentityUpdatesExposesTopLevelTargets(t *testing.T) {
	user := &model.User{ID: "user-1", Email: "user@example.com", Name: "Test User"}
	var output bytes.Buffer
	if err := SidebarIdentityUpdates(user).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	html := output.String()
	for _, id := range []string{"sidebar-avatar", "sidebar-user-name", "dropdown-user-label"} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Fatalf("identity updates do not contain %s: %s", id, html)
		}
	}
	if strings.Contains(html, "Settings</span>") {
		t.Fatalf("identity updates rendered the full dropdown: %s", html)
	}
}
