// Package api defines the stable JSON contract shared by the public CLI and
// the private goilerplate service.
package api

import "time"

const (
	PathGitHubLogin      = "/v1/auth/github"
	PathLogout           = "/v1/auth/logout"
	PathWhoAmI           = "/v1/account"
	PathGenerate         = "/v1/generate"
	PathUpdate           = "/v1/update"
	PathLicenseClaim     = "/v1/license/claim"
	PathLicenseClaimCode = "/v1/license/claim/code"
	PathAccountDelete    = "/v1/account"
	PathLicenses         = "/v1/licenses/"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GitHubLoginRequest struct {
	OAuthAccessToken string `json:"oauth_access_token"`
}

type GitHubLoginResponse struct {
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Account      Account   `json:"account"`
}

type BeginLicenseClaimRequest struct {
	PurchaseEmail string `json:"purchase_email"`
}

type ConfirmLicenseClaimRequest struct {
	Code string `json:"code"`
}

type WhoAmIResponse struct {
	Account  Account   `json:"account"`
	Licenses []License `json:"licenses"`
}

type Account struct {
	ID          string `json:"id"`
	GitHubLogin string `json:"github_login"`
	Email       string `json:"email"`
}

type License struct {
	ID     string        `json:"id"`
	Status LicenseStatus `json:"status"`
	Role   LicenseRole   `json:"role"`
}

type LicenseStatus string

const (
	LicenseStatusActive  LicenseStatus = "active"
	LicenseStatusRevoked LicenseStatus = "revoked"
)

type LicenseRole string

const (
	LicenseRoleOwner  LicenseRole = "owner"
	LicenseRoleMember LicenseRole = "member"
)

type LicenseMembersResponse struct {
	Members     []LicenseMember     `json:"members"`
	Invitations []LicenseInvitation `json:"invitations"`
}

type LicenseMember struct {
	UserID      string      `json:"user_id"`
	GitHubLogin string      `json:"github_login"`
	Email       string      `json:"email"`
	Role        LicenseRole `json:"role"`
}

type LicenseInvitation struct {
	ID        string      `json:"id"`
	Email     string      `json:"email"`
	Role      LicenseRole `json:"role"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type InviteLicenseMemberRequest struct {
	Email string      `json:"email"`
	Role  LicenseRole `json:"role"`
}

type InviteLicenseMemberResponse struct {
	Joined bool `json:"joined"`
}

type DeleteAccountRequest struct {
	ConfirmGitHubLogin string `json:"confirm_github_login"`
}

type GenerateRequest struct {
	TemplateVersion string            `json:"template_version,omitempty"`
	Answers         GenerationAnswers `json:"answers"`
}

const LockSchemaVersion = 1

type ProjectLock struct {
	SchemaVersion   int               `json:"schema_version"`
	TemplateVersion string            `json:"template_version"`
	Answers         GenerationAnswers `json:"answers"`
}

type GenerationAnswers struct {
	ProjectName string   `json:"project_name"`
	ModulePath  string   `json:"module_path"`
	Edition     string   `json:"edition"`
	Framework   string   `json:"framework"`
	Database    string   `json:"database"`
	Payment     string   `json:"payment"`
	Mail        string   `json:"mail"`
	Teams       bool     `json:"teams"`
	OAuth       []string `json:"oauth"`
	Storage     bool     `json:"storage"`
	Content     []string `json:"content"`
	Example     bool     `json:"example"`
}
