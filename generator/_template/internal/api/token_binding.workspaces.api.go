package api

import (
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
)

// visibleMemberships keeps only the workspace a personal API token is bound
// to. Session callers see every membership.
func visibleMemberships(r *http.Request, memberships []model.WorkspaceMembership) []model.WorkspaceMembership {
	token := ctxkeys.APIToken(r.Context())
	if token == nil {
		return memberships
	}
	visible := make([]model.WorkspaceMembership, 0, 1)
	for _, membership := range memberships {
		if membership.ID == token.WorkspaceID {
			visible = append(visible, membership)
		}
	}
	return visible
}
