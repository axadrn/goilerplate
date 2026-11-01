package api

import "{{.ModulePath}}/internal/model"

// Workspace is the public JSON shape of a workspace.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// NewWorkspace converts a workspace model into its public JSON shape.
func NewWorkspace(workspace *model.Workspace) Workspace {
	return Workspace{ID: workspace.ID, Name: workspace.Name, Slug: workspace.Slug}
}

// Membership is a workspace the user belongs to and the user's role in it.
type Membership struct {
	Workspace Workspace `json:"workspace"`
	Role      string    `json:"role"`
}

// NewMemberships converts workspace memberships into their public JSON shape.
func NewMemberships(memberships []model.WorkspaceMembership) []Membership {
	items := make([]Membership, 0, len(memberships))
	for index := range memberships {
		items = append(items, Membership{
			Workspace: NewWorkspace(&memberships[index].Workspace),
			Role:      memberships[index].Role,
		})
	}
	return items
}
