// Package api defines the stable JSON contract shared by the public CLI and
// the private Goilerplate service.
package api

import "time"

const (
	PathGitHubLogin = "/v1/auth/github"
	PathLogout      = "/v1/auth/logout"
	PathWhoAmI      = "/v1/account"
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

type WhoAmIResponse struct {
	Account            Account   `json:"account"`
	Licenses           []License `json:"licenses"`
	EffectiveLicenseID string    `json:"effective_license_id,omitempty"`
}

type Account struct {
	ID          string `json:"id"`
	GitHubLogin string `json:"github_login"`
	Email       string `json:"email"`
}

type License struct {
	ID         string        `json:"id"`
	Tier       LicenseTier   `json:"tier"`
	Status     LicenseStatus `json:"status"`
	Role       LicenseRole   `json:"role"`
	ValidUntil *time.Time    `json:"valid_until,omitempty"`
}

type LicenseTier string

const (
	LicenseTierFree          LicenseTier = "free"
	LicenseTierPaid          LicenseTier = "paid"
	LicenseTierGrandfathered LicenseTier = "grandfathered"
)

type LicenseStatus string

const (
	LicenseStatusPending LicenseStatus = "pending"
	LicenseStatusActive  LicenseStatus = "active"
	LicenseStatusExpired LicenseStatus = "expired"
	LicenseStatusRevoked LicenseStatus = "revoked"
)

type LicenseRole string

const (
	LicenseRoleOwner  LicenseRole = "owner"
	LicenseRoleMember LicenseRole = "member"
)
