package pages

import (
	"time"

	"{{.ModulePath}}/internal/model"
	selectcomp "{{.ModulePath}}/internal/ui/components/select"
)

// APITokensView is the data of the API tokens card in account settings.
type APITokensView struct {
	Tokens []model.APIToken
	// Workspaces are the workspaces a new token can be bound to. Projects
	// without workspaces leave it empty and the form has no workspace field.
	Workspaces []APITokenWorkspace
	// WorkspaceNames maps a token ID to the name of its workspace.
	WorkspaceNames map[string]string
	Form           APITokenForm
	Now            time.Time
}

// APITokenWorkspace is one choice of the workspace field.
type APITokenWorkspace struct {
	ID   string
	Name string
}

// APITokenForm is the state of the create form, with one message per invalid
// field in Errors.
type APITokenForm struct {
	Name        string
	Access      string
	Expiry      string
	WorkspaceID string
	Errors      map[string]string
}

const (
	// APITokenAccessRead is the form value for a read only token.
	APITokenAccessRead = "read"
	// APITokenAccessWrite is the form value for a token that may also change
	// data.
	APITokenAccessWrite = "write"
	// APITokenExpiryNever is the form value for a token that never expires.
	APITokenExpiryNever = "never"
)

// NewAPITokenForm returns an empty form: read only, 90 days, bound to the
// given workspace when there is one.
func NewAPITokenForm(workspaceID string) APITokenForm {
	return APITokenForm{Access: APITokenAccessRead, Expiry: "90", WorkspaceID: workspaceID}
}

var apiTokenAccessItems = []selectcomp.ItemData{
	{Value: APITokenAccessRead, Label: "Read only"},
	{Value: APITokenAccessWrite, Label: "Read and write"},
}

var apiTokenExpiryItems = []selectcomp.ItemData{
	{Value: "7", Label: "7 days"},
	{Value: "30", Label: "30 days"},
	{Value: "90", Label: "90 days"},
	{Value: "365", Label: "1 year"},
	{Value: APITokenExpiryNever, Label: "No expiration"},
}

func (v APITokensView) workspaceItems() []selectcomp.ItemData {
	items := make([]selectcomp.ItemData, 0, len(v.Workspaces))
	for _, workspace := range v.Workspaces {
		items = append(items, selectcomp.ItemData{Value: workspace.ID, Label: workspace.Name})
	}
	return items
}

func apiTokenAccessLabel(token model.APIToken) string {
	if token.CanWrite() {
		return "Read and write"
	}
	return "Read only"
}

func apiTokenLastUsed(token model.APIToken) string {
	if token.LastUsedAt == nil {
		return "Never used"
	}
	return "Last used " + token.LastUsedAt.Local().Format("Jan 2, 2006")
}

func apiTokenExpiry(token model.APIToken, now time.Time) string {
	if token.ExpiresAt == nil {
		return "No expiration"
	}
	if token.IsExpired(now) {
		return "Expired " + token.ExpiresAt.Local().Format("Jan 2, 2006")
	}
	return "Expires " + token.ExpiresAt.Local().Format("Jan 2, 2006")
}

// apiTokenInvalid is the aria-invalid value of a field with this error
// message. Styles match aria-invalid="true" only.
func apiTokenInvalid(message string) string {
	if message == "" {
		return "false"
	}
	return "true"
}
