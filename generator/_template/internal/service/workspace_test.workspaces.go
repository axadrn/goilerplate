package service

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestWorkspaceServiceInvitationFlowAndMultipleWorkspaces(t *testing.T) {
	database, workspaces, mailer := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "service-owner", Email: "owner@example.com", Name: "Owner"})
	invitee := createWorkspaceTestUser(t, database, "invited-user", "invited@example.com")

	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); !errors.Is(err, repository.ErrSeatLimit) {
		t.Fatalf("Invite() on full Free workspace error = %v, want ErrSeatLimit", err)
	}
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("upgrade owner workspace: %v", err)
	}
	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, invitee.Email, model.WorkspaceRoleMember); err != nil {
		t.Fatalf("Invite() with available seat error = %v", err)
	}
	token := invitationTokenFromLastEmail(t, mailer)
	owner.Subscription.PlanID = model.SubscriptionPlanFree
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("downgrade owner workspace: %v", err)
	}
	if _, err := workspaces.AcceptInvitation(token, invitee); !errors.Is(err, repository.ErrSeatLimit) {
		t.Fatalf("AcceptInvitation() after downgrade error = %v, want ErrSeatLimit", err)
	}
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("restore owner workspace plan: %v", err)
	}
	joined, err := workspaces.AcceptInvitation(token, invitee)
	if err != nil {
		t.Fatalf("AcceptInvitation() error = %v", err)
	}
	if joined.ID != owner.Workspace.ID {
		t.Fatalf("joined workspace = %q, want %q", joined.ID, owner.Workspace.ID)
	}

	member, err := repository.NewWorkspaceMemberRepository(database).ByWorkspaceAndUser(owner.Workspace.ID, invitee.ID)
	if err != nil || member.Role != model.WorkspaceRoleMember {
		t.Fatalf("accepted member = %#v, error = %v", member, err)
	}

	stored := &model.WorkspaceInvitation{}
	if err := database.Get(stored, `SELECT * FROM workspace_invitations WHERE workspace_id = $1 AND email = $2`, owner.Workspace.ID, invitee.Email); err != nil {
		t.Fatalf("load invitation: %v", err)
	}
	if stored.TokenHash == token || strings.Contains(stored.TokenHash, token) || len(stored.TokenHash) != 64 {
		t.Fatalf("stored invitation token is not a SHA-256 hash: %q", stored.TokenHash)
	}

	multi := testutil.SeedUser(t, database, testutil.UserOptions{ID: "multi-user", Email: "multi@example.com", Name: "Multi User"})
	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, multi.User.Email, model.WorkspaceRoleAdmin); err != nil {
		t.Fatalf("Invite() multi-workspace user error = %v", err)
	}
	if _, err := workspaces.AcceptInvitation(invitationTokenFromLastEmail(t, mailer), multi.User); err != nil {
		t.Fatalf("AcceptInvitation() multi-workspace user error = %v", err)
	}
	workspace_members, err := workspaces.ListForUser(multi.User.ID)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(workspace_members) != 2 {
		t.Fatalf("multi-workspace member count = %d, want 2", len(workspace_members))
	}

	if _, _, err := workspaces.BySlugForUser(multi.Workspace.Slug, owner.User.ID); !errors.Is(err, repository.ErrWorkspaceNotFound) {
		t.Fatalf("cross-workspace access error = %v, want ErrWorkspaceNotFound", err)
	}
	if err := workspaces.RecordBillingAction(owner.Workspace.ID, owner.User.ID, "billing.checkout_started"); err != nil {
		t.Fatalf("RecordBillingAction() error = %v", err)
	}

	events, err := workspaces.AuditEvents(owner.Workspace.ID)
	if err != nil {
		t.Fatalf("AuditEvents() error = %v", err)
	}
	if !hasAuditAction(events, "workspace.invitation.sent") || !hasAuditAction(events, "workspace.invitation.accepted") || !hasAuditAction(events, "billing.checkout_started") {
		t.Fatalf("audit events = %#v", events)
	}
}

func TestWorkspaceServiceRoleBoundaries(t *testing.T) {
	database, workspaces, mailer := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "roles-owner", Email: "owner@example.com", Name: "Owner"})
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	if err := repository.NewSubscriptionRepository(database).Update(owner.Subscription); err != nil {
		t.Fatalf("upgrade owner workspace: %v", err)
	}
	admin := createWorkspaceTestUser(t, database, "roles-admin", "admin@example.com")
	member := createWorkspaceTestUser(t, database, "roles-member", "member@example.com")

	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, admin.Email, model.WorkspaceRoleAdmin); err != nil {
		t.Fatalf("invite admin: %v", err)
	}
	if _, err := workspaces.AcceptInvitation(invitationTokenFromLastEmail(t, mailer), admin); err != nil {
		t.Fatalf("accept admin: %v", err)
	}
	if err := workspaces.Invite(owner.WorkspaceMember, owner.Workspace, member.Email, model.WorkspaceRoleMember); err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if _, err := workspaces.AcceptInvitation(invitationTokenFromLastEmail(t, mailer), member); err != nil {
		t.Fatalf("accept member: %v", err)
	}

	members := repository.NewWorkspaceMemberRepository(database)
	adminMembership, _ := members.ByWorkspaceAndUser(owner.Workspace.ID, admin.ID)
	memberMembership, _ := members.ByWorkspaceAndUser(owner.Workspace.ID, member.ID)

	ownerMembers, err := workspaces.Members(owner.WorkspaceMember)
	if err != nil {
		t.Fatalf("owner Members() error = %v", err)
	}
	if len(ownerMembers) != 3 {
		t.Fatalf("owner Members() count = %d, want 3", len(ownerMembers))
	}
	for _, listedMember := range ownerMembers {
		if listedMember.Email == "" {
			t.Fatalf("owner Members() hid email for %q", listedMember.Name)
		}
	}
	managedMembers, err := workspaces.Members(adminMembership)
	if err != nil {
		t.Fatalf("admin Members() error = %v", err)
	}
	if len(managedMembers) != 3 {
		t.Fatalf("admin Members() count = %d, want 3", len(managedMembers))
	}
	for _, listedMember := range managedMembers {
		if listedMember.Email == "" {
			t.Fatalf("admin Members() hid email for %q", listedMember.Name)
		}
	}
	visibleMembers, err := workspaces.Members(memberMembership)
	if err != nil {
		t.Fatalf("member Members() error = %v", err)
	}
	if len(visibleMembers) != 3 {
		t.Fatalf("member Members() count = %d, want 3", len(visibleMembers))
	}
	for _, listedMember := range visibleMembers {
		if listedMember.Name == "" || listedMember.Role == "" {
			t.Fatalf("member Members() omitted name or role: %#v", listedMember)
		}
		if listedMember.Email != "" {
			t.Fatalf("member Members() exposed email %q", listedMember.Email)
		}
	}

	if err := workspaces.UpdateName(adminMembership, owner.Workspace, "Admin renamed workspace"); err != nil {
		t.Fatalf("admin UpdateName() error = %v", err)
	}
	if err := workspaces.UpdateName(memberMembership, owner.Workspace, "Member renamed workspace"); !errors.Is(err, ErrWorkspacePermission) {
		t.Fatalf("member UpdateName() error = %v, want ErrWorkspacePermission", err)
	}
	other := testutil.SeedUser(t, database, testutil.UserOptions{ID: "roles-other", Email: "other@example.com", Name: "Other"})
	if err := workspaces.UpdateName(owner.WorkspaceMember, other.Workspace, "Cross tenant rename"); !errors.Is(err, ErrWorkspacePermission) {
		t.Fatalf("cross-tenant UpdateName() error = %v, want ErrWorkspacePermission", err)
	}
	if err := workspaces.ChangeRole(adminMembership, owner.Workspace.ID, owner.WorkspaceMember.ID, model.WorkspaceRoleMember); !errors.Is(err, ErrWorkspacePermission) {
		t.Fatalf("admin owner-role change error = %v, want ErrWorkspacePermission", err)
	}
	if err := workspaces.ChangeRole(owner.WorkspaceMember, owner.Workspace.ID, adminMembership.ID, model.WorkspaceRoleOwner); err != nil {
		t.Fatalf("owner promotion error = %v", err)
	}
	if err := workspaces.ChangeRole(owner.WorkspaceMember, owner.Workspace.ID, owner.WorkspaceMember.ID, model.WorkspaceRoleMember); err != nil {
		t.Fatalf("owner role change with another owner error = %v", err)
	}
}

func TestWorkspaceNameRejectsControlCharacters(t *testing.T) {
	database, workspaces, _ := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "name-owner", Name: "Owner"})
	for _, name := range []string{"Bad\nWorkspace", "Bad\rWorkspace", "Bad\tWorkspace"} {
		if err := workspaces.UpdateName(owner.WorkspaceMember, owner.Workspace, name); !errors.Is(err, ErrInvalidWorkspace) {
			t.Fatalf("UpdateName(%q) error = %v, want ErrInvalidWorkspace", name, err)
		}
	}
	if _, err := workspaces.CreateDefault(owner.User.ID, "Bad\nName"); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("CreateDefault() error = %v, want ErrInvalidWorkspace", err)
	}
}

func TestWorkspaceDeletionRequiresBillingToEnd(t *testing.T) {
	database, workspaces, _ := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "delete-billing-owner", Name: "Owner"})
	subscriptions := repository.NewSubscriptionRepository(database)
	providerSubscriptionID := "provider-subscription-1"
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	owner.Subscription.Provider = model.ProviderStripe
	owner.Subscription.ProviderSubscriptionID = &providerSubscriptionID
	if err := subscriptions.Update(owner.Subscription); err != nil {
		t.Fatalf("activate paid subscription: %v", err)
	}

	if err := workspaces.Delete(owner.WorkspaceMember, owner.Workspace); !errors.Is(err, ErrActiveSubscription) {
		t.Fatalf("Delete() active subscription error = %v, want ErrActiveSubscription", err)
	}
	if _, err := repository.NewWorkspaceRepository(database).ByID(owner.Workspace.ID); err != nil {
		t.Fatalf("active workspace was deleted: %v", err)
	}

	ended := time.Now().Add(-time.Hour)
	owner.Subscription.Status = model.SubscriptionStatusCancelled
	owner.Subscription.CurrentPeriodEnd = &ended
	if err := subscriptions.Update(owner.Subscription); err != nil {
		t.Fatalf("expire paid subscription: %v", err)
	}
	if err := workspaces.Delete(owner.WorkspaceMember, owner.Workspace); !errors.Is(err, ErrActiveSubscription) {
		t.Fatalf("Delete() expired local access with provider subscription error = %v, want ErrActiveSubscription", err)
	}
	owner.Subscription.ProviderSubscriptionID = nil
	if err := subscriptions.Update(owner.Subscription); err != nil {
		t.Fatalf("clear provider subscription: %v", err)
	}
	if err := workspaces.Delete(owner.WorkspaceMember, owner.Workspace); err != nil {
		t.Fatalf("Delete() after provider subscription ended error = %v", err)
	}
	if _, err := repository.NewWorkspaceRepository(database).ByID(owner.Workspace.ID); !errors.Is(err, repository.ErrWorkspaceNotFound) {
		t.Fatalf("deleted workspace lookup error = %v, want ErrWorkspaceNotFound", err)
	}
	events, err := repository.NewAdminRepository(database).ListAuditEvents(10)
	if err != nil {
		t.Fatalf("ListAuditEvents() error = %v", err)
	}
	if len(events) == 0 || events[0].Action != "workspace.deleted" || events[0].WorkspaceID == nil || *events[0].WorkspaceID != owner.Workspace.ID {
		t.Fatalf("retained workspace deletion event = %#v", events)
	}
}

func TestPendingCheckoutBlocksWorkspaceDeletion(t *testing.T) {
	database, workspaces, _ := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "pending-checkout-owner", Name: "Owner"})
	subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	pendingUntil := time.Now().Add(time.Hour)
	if _, _, _, err := subscriptions.BeginCheckout(owner.Workspace.ID, model.ProviderStripe, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, pendingUntil.Add(-time.Minute), pendingUntil); err != nil {
		t.Fatalf("BeginCheckout() error = %v", err)
	}
	if err := workspaces.Delete(owner.WorkspaceMember, owner.Workspace); !errors.Is(err, ErrActiveSubscription) {
		t.Fatalf("Delete() pending checkout error = %v, want ErrActiveSubscription", err)
	}
	if _, err := database.Exec(`UPDATE subscriptions SET checkout_pending_until = $1 WHERE workspace_id = $2`, time.Now().Add(-time.Minute), owner.Workspace.ID); err != nil {
		t.Fatalf("expire checkout marker: %v", err)
	}
	if err := workspaces.Delete(owner.WorkspaceMember, owner.Workspace); err != nil {
		t.Fatalf("Delete() after checkout failure error = %v", err)
	}
}

func TestConcurrentPaidWebhookAndDeletionKeepWorkspaceRecoverable(t *testing.T) {
	database, workspaces, _ := workspaceTestService(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "webhook-delete-owner", Name: "Owner"})
	subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	pendingUntil := time.Now().Add(time.Hour)
	if _, _, _, err := subscriptions.BeginCheckout(owner.Workspace.ID, model.ProviderStripe, model.SubscriptionPlanPro, model.SubscriptionIntervalMonthly, pendingUntil.Add(-time.Minute), pendingUntil); err != nil {
		t.Fatalf("BeginCheckout() error = %v", err)
	}
	providerSubscriptionID := "concurrent-provider-subscription"
	owner.Subscription.PlanID = model.SubscriptionPlanPro
	owner.Subscription.Provider = model.ProviderPolar
	owner.Subscription.ProviderSubscriptionID = &providerSubscriptionID
	owner.Subscription.ClearCheckout()

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		results <- subscriptions.UpdateSubscription(owner.Subscription)
	}()
	go func() {
		defer wait.Done()
		<-start
		results <- workspaces.Delete(owner.WorkspaceMember, owner.Workspace)
	}()
	close(start)
	wait.Wait()
	close(results)

	successes := 0
	billingBlocks := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrActiveSubscription):
			billingBlocks++
		default:
			t.Fatalf("concurrent billing result = %v", err)
		}
	}
	if successes != 1 || billingBlocks != 1 {
		t.Fatalf("successes = %d, billing blocks = %d", successes, billingBlocks)
	}
	stored, err := repository.NewSubscriptionRepository(database).ByWorkspaceID(owner.Workspace.ID)
	if err != nil || stored.PlanID != model.SubscriptionPlanPro || valueOrEmpty(stored.ProviderSubscriptionID) != providerSubscriptionID {
		t.Fatalf("stored subscription = %#v, error = %v", stored, err)
	}
}

func workspaceTestService(t *testing.T) (*sqlx.DB, *WorkspaceService, *testutil.MemoryMailer) {
	t.Helper()
	database := testutil.NewDatabase(t)
	users := repository.NewUserRepository(database)
	subscriptions := NewSubscriptionService(repository.NewSubscriptionRepository(database), repository.NewAdminRepository(database))
	mailer := testutil.NewMemoryMailer()
	email := NewEmailService(mailer, "https://app.example.com", "Test App")
	workspaces := NewWorkspaceService(
		repository.NewWorkspaceRepository(database),
		repository.NewWorkspaceMemberRepository(database),
		repository.NewWorkspaceInvitationRepository(database),
		repository.NewWorkspaceAuditRepository(database),
		users,
		subscriptions,
		email,
	)
	return database, workspaces, mailer
}

func createWorkspaceTestUser(t *testing.T, database *sqlx.DB, id, email string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{ID: id, Email: email, Name: id, CreatedAt: now}
	if err := repository.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func invitationTokenFromLastEmail(t *testing.T, mailer *testutil.MemoryMailer) string {
	t.Helper()
	messages := mailer.Messages()
	if len(messages) == 0 {
		t.Fatal("no invitation email was sent")
	}
	body := messages[len(messages)-1].Text
	marker := "/workspaces/invitations/"
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("invitation URL missing from email: %q", body)
	}
	token := strings.Fields(body[start+len(marker):])[0]
	if token == "" {
		t.Fatal("invitation token is empty")
	}
	return token
}

func hasAuditAction(events []model.WorkspaceAuditEvent, action string) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func TestWorkspaceSlugIsShortAndReadable(t *testing.T) {
	for name, want := range map[string]string{
		"Ada Lovelace":                  "ada-lovelace",
		"Acme GmbH & Co. KG":            "acme-gmbh-co-kg",
		"Ärzte Zürich":                  "arzte-zurich",
		"Straße 42":                     "strasse-42",
		"  --Blog--  ":                  "blog",
		"東京":                            "team",
		"x":                             "team",
		"Settings":                      "team",
		"github":                        "team",
		strings.Repeat("abcdefghij", 6): "abcdefghijabcdefghijabcdefghijabcdefghij",
	} {
		if got := workspaceSlug(name); got != want {
			t.Errorf("workspaceSlug(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestWorkspaceSlugsTakeASuffixOnlyWhenTaken(t *testing.T) {
	database, workspaces, _ := workspaceTestService(t)
	owner := createWorkspaceTestUser(t, database, "slug-owner", "slugs@example.com")
	want := []string{"acme", "acme-2", "acme-3"}
	for _, slug := range want {
		workspace, err := workspaces.Create(owner.ID, "Acme")
		if err != nil {
			t.Fatal(err)
		}
		if workspace.Slug != slug {
			t.Fatalf("slug = %q, want %q", workspace.Slug, slug)
		}
	}
	personal, err := workspaces.CreateDefault(owner.ID, "Ada Lovelace")
	if err != nil {
		t.Fatal(err)
	}
	if personal.Slug != "ada-lovelace" || personal.Name != "Ada Lovelace's Workspace" {
		t.Fatalf("default workspace = %q %q", personal.Name, personal.Slug)
	}
}
