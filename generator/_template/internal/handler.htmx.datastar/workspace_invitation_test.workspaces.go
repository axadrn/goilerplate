package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/internal/testutil"
)

func TestInvitationShowPreservesGuestAndIncompleteOnboardingFlow(t *testing.T) {
	for _, test := range []struct {
		name       string
		signedIn   bool
		redirectTo string
	}{
		{name: "guest", redirectTo: "/auth"},
		{name: "authenticated incomplete profile", signedIn: true, redirectTo: "/auth/onboarding"},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, workspaces, owner, mailer := invitationHandlerTestSetup(t)
			invitee := createHandlerTestUser(t, database, "show-invitee", "show@example.com", "")
			if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); err != nil {
				t.Fatalf("Invite() error = %v", err)
			}
			token := invitationToken(t, mailer)
			request := httptest.NewRequest(http.MethodGet, "/workspaces/invitations/"+token, nil)
			request.SetPathValue("token", token)
			request = request.WithContext(ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{}))
			if test.signedIn {
				request = request.WithContext(ctxkeys.WithUser(request.Context(), invitee))
			}
			recorder := httptest.NewRecorder()

			NewWorkspaceInvitationHandler(workspaces, &config.Config{}).Show(recorder, request)

			if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != test.redirectTo {
				t.Fatalf("status = %d, location = %q", recorder.Code, recorder.Header().Get("Location"))
			}
			if pendingCookie(recorder) == nil || pendingCookie(recorder).Value != token {
				t.Fatalf("pending invitation cookie = %#v", pendingCookie(recorder))
			}
		})
	}
}

func TestInvitationTerminalOutcomesClearPendingCookie(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, database *sqlx.DB, workspaces *service.WorkspaceService, owner testutil.UserFixture, invitee *model.User, invitationID, token string)
		show         bool
		invalidToken bool
	}{
		{
			name:         "invalid",
			show:         true,
			invalidToken: true,
			setup: func(*testing.T, *sqlx.DB, *service.WorkspaceService, testutil.UserFixture, *model.User, string, string) {
			},
		},
		{
			name: "expired",
			show: true,
			setup: func(t *testing.T, database *sqlx.DB, _ *service.WorkspaceService, _ testutil.UserFixture, _ *model.User, invitationID, _ string) {
				_, err := database.Exec(`UPDATE workspace_invitations SET expires_at = $1 WHERE id = $2`, time.Now().Add(-time.Hour), invitationID)
				if err != nil {
					t.Fatalf("expire invitation: %v", err)
				}
			},
		},
		{
			name: "revoked",
			show: true,
			setup: func(t *testing.T, _ *sqlx.DB, workspaces *service.WorkspaceService, owner testutil.UserFixture, _ *model.User, invitationID, _ string) {
				if err := workspaces.RevokeInvitation(owner.WorkspaceMember, owner.Workspace.ID, invitationID); err != nil {
					t.Fatalf("RevokeInvitation() error = %v", err)
				}
			},
		},
		{
			name: "used",
			show: true,
			setup: func(t *testing.T, database *sqlx.DB, workspaces *service.WorkspaceService, owner testutil.UserFixture, invitee *model.User, _, token string) {
				owner.Subscription.PlanID = model.SubscriptionPlanPro
				if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
					t.Fatalf("upgrade subscription: %v", err)
				}
				if _, err := workspaces.AcceptInvitation(token, invitee); err != nil {
					t.Fatalf("AcceptInvitation() error = %v", err)
				}
			},
		},
		{
			name: "already member",
			setup: func(t *testing.T, database *sqlx.DB, _ *service.WorkspaceService, owner testutil.UserFixture, invitee *model.User, _, _ string) {
				_, err := database.Exec(`INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at) VALUES ($1, $2, $3, $4, $5)`, "existing-member", owner.Workspace.ID, invitee.ID, model.WorkspaceRoleMember, time.Now())
				if err != nil {
					t.Fatalf("create existing member: %v", err)
				}
			},
		},
		{
			name: "wrong email",
			setup: func(t *testing.T, database *sqlx.DB, _ *service.WorkspaceService, _ testutil.UserFixture, invitee *model.User, _, _ string) {
				other := createHandlerTestUser(t, database, "wrong-email-user", "wrong@example.com", "Wrong User")
				*invitee = *other
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, workspaces, owner, mailer := invitationHandlerTestSetup(t)
			invitee := createHandlerTestUser(t, database, "terminal-invitee", "terminal@example.com", "Invitee")
			if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); err != nil {
				t.Fatalf("Invite() error = %v", err)
			}
			token := invitationToken(t, mailer)
			var invitationID string
			if err := database.Get(&invitationID, `SELECT id FROM workspace_invitations WHERE workspace_id = $1 AND email = $2`, owner.Workspace.ID, "terminal@example.com"); err != nil {
				t.Fatalf("load invitation ID: %v", err)
			}
			test.setup(t, database, workspaces, owner, invitee, invitationID, token)
			if test.invalidToken {
				token = "invalid-invitation-token"
			}

			method := http.MethodPost
			if test.show {
				method = http.MethodGet
			}
			request := httptest.NewRequest(method, "/workspaces/invitations/"+token, nil)
			request.SetPathValue("token", token)
			request.AddCookie(&http.Cookie{Name: "pending_invitation", Value: token})
			ctx := ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{})
			ctx = ctxkeys.WithUser(ctx, invitee)
			request = request.WithContext(ctx)
			recorder := httptest.NewRecorder()
			handler := NewWorkspaceInvitationHandler(workspaces, &config.Config{})
			if test.show {
				handler.Show(recorder, request)
			} else {
				handler.Accept(recorder, request)
			}
			if cookie := pendingCookie(recorder); cookie == nil || cookie.MaxAge >= 0 {
				t.Fatalf("pending invitation cookie was not cleared: %#v", cookie)
			}
		})
	}
}

func TestInvitationSeatLimitClearsPendingCookieForRecovery(t *testing.T) {
	database, workspaces, owner, mailer := invitationHandlerTestSetup(t)
	invitee := createHandlerTestUser(t, database, "seat-invitee", "seat@example.com", "Invitee")
	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	token := invitationToken(t, mailer)
	owner.Subscription.PlanID = model.SubscriptionPlanFree
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("downgrade subscription: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/workspaces/invitations/"+token, nil)
	request.SetPathValue("token", token)
	request.AddCookie(&http.Cookie{Name: "pending_invitation", Value: token})
	ctx := ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{})
	request = request.WithContext(ctxkeys.WithUser(ctx, invitee))
	recorder := httptest.NewRecorder()

	NewWorkspaceInvitationHandler(workspaces, &config.Config{}).Accept(recorder, request)

	if cookie := pendingCookie(recorder); cookie == nil || cookie.MaxAge >= 0 {
		t.Fatalf("seat limit did not clear pending cookie: %#v", cookie)
	}
}

func TestOnboardingValidatesPendingInvitationBeforeSkippingWorkspace(t *testing.T) {
	for _, test := range []struct {
		name            string
		validInvitation bool
		wantMemberships int
		wantPrefix      string
	}{
		{name: "valid invitation", validInvitation: true, wantMemberships: 0, wantPrefix: "/workspaces/invitations/"},
		{name: "expired invitation", wantMemberships: 1, wantPrefix: "/app/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, workspaces, owner, mailer := invitationHandlerTestSetup(t)
			invitee := createHandlerTestUser(t, database, "onboarding-invitee", "onboarding@example.com", "")
			if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); err != nil {
				t.Fatalf("Invite() error = %v", err)
			}
			token := invitationToken(t, mailer)
			if !test.validInvitation {
				if _, err := database.Exec(`UPDATE workspace_invitations SET expires_at = $1 WHERE workspace_id = $2 AND email = $3`, time.Now().Add(-time.Hour), owner.Workspace.ID, invitee.Email); err != nil {
					t.Fatalf("expire invitation: %v", err)
				}
			}
			email := service.NewEmailService(testutil.NewMemoryMailer(), "https://app.example.com", "Test App")
			auth := service.NewAuthService(repository.NewUserRepository(database), nil, nil, workspaces, email, false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour)
			handler := NewAuthHandler(auth, nil, workspaces, &config.Config{})
			form := url.Values{"name": {"Invited User"}}
			request := httptest.NewRequest(http.MethodPost, "/auth/onboarding", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(&http.Cookie{Name: "pending_invitation", Value: token})
			ctx := ctxkeys.WithRenderConfig(request.Context(), config.RenderConfig{})
			request = request.WithContext(ctxkeys.WithUser(ctx, invitee))
			recorder := httptest.NewRecorder()

			handler.CompleteOnboarding(recorder, request)

			if recorder.Code != http.StatusSeeOther || !strings.HasPrefix(recorder.Header().Get("Location"), test.wantPrefix) {
				t.Fatalf("status = %d, location = %q", recorder.Code, recorder.Header().Get("Location"))
			}
			members, err := workspaces.ListForUser(invitee.ID)
			if err != nil {
				t.Fatalf("ListForUser() error = %v", err)
			}
			if len(members) != test.wantMemberships {
				t.Fatalf("members = %d, want %d", len(members), test.wantMemberships)
			}
		})
	}
}

func invitationHandlerTestSetup(t *testing.T) (*sqlx.DB, *service.WorkspaceService, testutil.UserFixture, *testutil.MemoryMailer) {
	t.Helper()
	database := testutil.NewDatabase(t)
	users := repository.NewUserRepository(database)
	mailer := testutil.NewMemoryMailer()
	email := service.NewEmailService(mailer, "https://app.example.com", "Test App")
	workspaces := service.NewWorkspaceService(
		repository.NewWorkspaceRepository(database),
		repository.NewWorkspaceMemberRepository(database),
		repository.NewWorkspaceInvitationRepository(database),
		repository.NewWorkspaceAuditRepository(database),
		users,
		service.NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database)),
		email,
	)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "handler-owner", Email: "owner@example.com", Name: "Owner"})
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("upgrade handler workspace: %v", err)
	}
	return database, workspaces, owner, mailer
}

func createHandlerTestUser(t *testing.T, database *sqlx.DB, id, email, name string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{ID: id, Email: email, Name: name, CreatedAt: now}
	if err := repository.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func invitationToken(t *testing.T, mailer *testutil.MemoryMailer) string {
	t.Helper()
	messages := mailer.Messages()
	if len(messages) == 0 {
		t.Fatal("no invitation email")
	}
	marker := "/workspaces/invitations/"
	start := strings.Index(messages[len(messages)-1].Text, marker)
	if start < 0 {
		t.Fatal("invitation token missing from email")
	}
	return strings.Fields(messages[len(messages)-1].Text[start+len(marker):])[0]
}

func pendingCookie(recorder *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "pending_invitation" {
			return cookie
		}
	}
	return nil
}
