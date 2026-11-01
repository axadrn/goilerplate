package repository_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestWorkspaceQueriesEnforceMembershipScope(t *testing.T) {
	database := testutil.NewDatabase(t)
	alice := testutil.SeedUser(t, database, testutil.UserOptions{ID: "alice", Name: "Alice"})
	bob := testutil.SeedUser(t, database, testutil.UserOptions{ID: "bob", Name: "Bob"})
	workspaceRepository := repository.NewWorkspaceRepository(database)

	if _, err := workspaceRepository.BySlugForUser(alice.Workspace.Slug, bob.User.ID); !errors.Is(err, repository.ErrWorkspaceNotFound) {
		t.Fatalf("cross-workspace lookup error = %v, want ErrWorkspaceNotFound", err)
	}

	if _, err := database.Exec(`
		INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "member-bob-alice", alice.Workspace.ID, bob.User.ID, model.WorkspaceRoleMember, time.Now()); err != nil {
		t.Fatalf("add second member: %v", err)
	}

	workspaces, err := workspaceRepository.ListForUser(bob.User.ID)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("workspace count = %d, want 2", len(workspaces))
	}
	if _, err := workspaceRepository.BySlugForUser(alice.Workspace.Slug, bob.User.ID); err != nil {
		t.Fatalf("member workspace lookup: %v", err)
	}
}

func TestWorkspaceMemberRepositoryProtectsLastOwner(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "owner", Name: "Owner"})
	workspaceMemberRepository := repository.NewWorkspaceMemberRepository(database)

	if err := workspaceMemberRepository.UpdateRole(owner.Workspace.ID, owner.User.ID, owner.WorkspaceMember.ID, model.WorkspaceRoleMember, testAudit(owner.Workspace.ID, owner.User.ID, "change-last-owner-role")); !errors.Is(err, repository.ErrLastOwner) {
		t.Fatalf("change last owner role error = %v, want ErrLastOwner", err)
	}
	if err := workspaceMemberRepository.Delete(owner.Workspace.ID, owner.User.ID, owner.WorkspaceMember.ID, testAudit(owner.Workspace.ID, owner.User.ID, "remove-last-owner")); !errors.Is(err, repository.ErrLastOwner) {
		t.Fatalf("delete last owner error = %v, want ErrLastOwner", err)
	}

	second := createBareUser(t, database, "second-owner", "second@example.com")
	if _, err := database.Exec(`
		INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "member-second-owner", owner.Workspace.ID, second.ID, model.WorkspaceRoleOwner, time.Now()); err != nil {
		t.Fatalf("add second owner: %v", err)
	}
	if err := workspaceMemberRepository.UpdateRole(owner.Workspace.ID, second.ID, owner.WorkspaceMember.ID, model.WorkspaceRoleMember, testAudit(owner.Workspace.ID, second.ID, "owner-role-changed")); err != nil {
		t.Fatalf("change owner role with second owner: %v", err)
	}
}

func TestAdminCannotRaceOwnerPromotion(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		database := testutil.NewDatabase(t)
		owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: fmt.Sprintf("race-owner-%d", iteration), Name: "Owner"})
		admin := createBareUser(t, database, fmt.Sprintf("race-admin-%d", iteration), fmt.Sprintf("admin-%d@example.com", iteration))
		target := createBareUser(t, database, fmt.Sprintf("race-target-%d", iteration), fmt.Sprintf("target-%d@example.com", iteration))
		adminMembershipID := fmt.Sprintf("race-admin-member-%d", iteration)
		targetMembershipID := fmt.Sprintf("race-target-member-%d", iteration)
		insertMembership(t, database, adminMembershipID, owner.Workspace.ID, admin.ID, model.WorkspaceRoleAdmin)
		insertMembership(t, database, targetMembershipID, owner.Workspace.ID, target.ID, model.WorkspaceRoleMember)
		members := repository.NewWorkspaceMemberRepository(database)

		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			results <- members.UpdateRole(owner.Workspace.ID, owner.User.ID, targetMembershipID, model.WorkspaceRoleOwner, testAudit(owner.Workspace.ID, owner.User.ID, "owner-promotes"))
		}()
		go func() {
			<-start
			results <- members.UpdateRole(owner.Workspace.ID, admin.ID, targetMembershipID, model.WorkspaceRoleAdmin, testAudit(owner.Workspace.ID, admin.ID, "admin-changes"))
		}()
		close(start)

		for range 2 {
			err := <-results
			if err != nil && !errors.Is(err, repository.ErrWorkspaceMemberPermission) {
				t.Fatalf("iteration %d concurrent role error = %v", iteration, err)
			}
		}
		stored, err := members.ByIDForWorkspace(targetMembershipID, owner.Workspace.ID)
		if err != nil || stored.Role != model.WorkspaceRoleOwner {
			t.Fatalf("iteration %d target = %#v, error = %v", iteration, stored, err)
		}
	}
}

func TestAdminCannotRaceOwnerPromotionAndRemoval(t *testing.T) {
	for iteration := 0; iteration < 10; iteration++ {
		database := testutil.NewDatabase(t)
		owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: fmt.Sprintf("remove-owner-%d", iteration), Name: "Owner"})
		admin := createBareUser(t, database, fmt.Sprintf("remove-admin-%d", iteration), fmt.Sprintf("remove-admin-%d@example.com", iteration))
		target := createBareUser(t, database, fmt.Sprintf("remove-target-%d", iteration), fmt.Sprintf("remove-target-%d@example.com", iteration))
		targetMembershipID := fmt.Sprintf("remove-target-member-%d", iteration)
		insertMembership(t, database, fmt.Sprintf("remove-admin-member-%d", iteration), owner.Workspace.ID, admin.ID, model.WorkspaceRoleAdmin)
		insertMembership(t, database, targetMembershipID, owner.Workspace.ID, target.ID, model.WorkspaceRoleMember)
		members := repository.NewWorkspaceMemberRepository(database)

		type result struct {
			action string
			err    error
		}
		start := make(chan struct{})
		results := make(chan result, 2)
		go func() {
			<-start
			results <- result{"promote", members.UpdateRole(owner.Workspace.ID, owner.User.ID, targetMembershipID, model.WorkspaceRoleOwner, testAudit(owner.Workspace.ID, owner.User.ID, "owner-promotes-remove-target"))}
		}()
		go func() {
			<-start
			results <- result{"remove", members.Delete(owner.Workspace.ID, admin.ID, targetMembershipID, testAudit(owner.Workspace.ID, admin.ID, "admin-removes"))}
		}()
		close(start)

		outcomes := map[string]error{}
		for range 2 {
			outcome := <-results
			outcomes[outcome.action] = outcome.err
		}
		switch {
		case outcomes["promote"] == nil && errors.Is(outcomes["remove"], repository.ErrWorkspaceMemberPermission):
			stored, err := members.ByIDForWorkspace(targetMembershipID, owner.Workspace.ID)
			if err != nil || stored.Role != model.WorkspaceRoleOwner {
				t.Fatalf("iteration %d promoted target = %#v, error = %v", iteration, stored, err)
			}
		case errors.Is(outcomes["promote"], repository.ErrWorkspaceMemberNotFound) && outcomes["remove"] == nil:
			if _, err := members.ByIDForWorkspace(targetMembershipID, owner.Workspace.ID); !errors.Is(err, repository.ErrWorkspaceMemberNotFound) {
				t.Fatalf("iteration %d removed target lookup error = %v", iteration, err)
			}
		default:
			t.Fatalf("iteration %d promote error = %v, remove error = %v", iteration, outcomes["promote"], outcomes["remove"])
		}
	}
}

func TestConcurrentInvitationAcceptanceCannotOversubscribeSeats(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "seat-owner", Name: "Owner"})
	first := createBareUser(t, database, "first-invitee", "first@example.com")
	second := createBareUser(t, database, "second-invitee", "second@example.com")
	workspaceInvitationRepository := repository.NewWorkspaceInvitationRepository(database)
	now := time.Now()
	if _, err := database.Exec(`UPDATE subscriptions SET plan_id = $1 WHERE workspace_id = $2`, model.SubscriptionPlanPro, owner.Workspace.ID); err != nil {
		t.Fatalf("set seat plan: %v", err)
	}
	for i := 1; i <= 3; i++ {
		userID := fmt.Sprintf("existing-member-%d", i)
		existing := createBareUser(t, database, userID, userID+"@example.com")
		if _, err := database.Exec(`
			INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, "member-"+userID, owner.Workspace.ID, existing.ID, model.WorkspaceRoleMember, now); err != nil {
			t.Fatalf("add existing member: %v", err)
		}
	}

	firstToken := "first-secret-token"
	secondToken := "second-secret-token"
	createInvitation(t, workspaceInvitationRepository, owner, "invitation-first", first.Email, firstToken, now)
	createInvitation(t, workspaceInvitationRepository, owner, "invitation-second", second.Email, secondToken, now)

	type attempt struct {
		err error
	}
	results := make(chan attempt, 2)
	var start sync.WaitGroup
	start.Add(1)
	for _, candidate := range []struct {
		token, userID, email, memberID, auditID string
	}{
		{firstToken, first.ID, first.Email, "accepted-first", "accepted-audit-first"},
		{secondToken, second.ID, second.Email, "accepted-second", "accepted-audit-second"},
	} {
		candidate := candidate
		go func() {
			start.Wait()
			_, err := workspaceInvitationRepository.Accept(invitationHash(candidate.token), candidate.userID, candidate.email, candidate.memberID, candidate.auditID, time.Now())
			results <- attempt{err: err}
		}()
	}
	start.Done()

	successes := 0
	seatFailures := 0
	for range 2 {
		result := <-results
		switch {
		case result.err == nil:
			successes++
		case errors.Is(result.err, repository.ErrSeatLimit):
			seatFailures++
		default:
			t.Fatalf("unexpected acceptance error: %v", result.err)
		}
	}
	if successes != 1 || seatFailures != 1 {
		t.Fatalf("successes = %d, seat failures = %d, want 1 and 1", successes, seatFailures)
	}

	var members int
	if err := database.Get(&members, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1`, owner.Workspace.ID); err != nil {
		t.Fatalf("count members: %v", err)
	}
	if members != 5 {
		t.Fatalf("member count = %d, want 5", members)
	}
}

func TestInvitationAcceptanceUsesPlanInsideWorkspaceLock(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "downgrade-owner", Name: "Owner"})
	invitee := createBareUser(t, database, "downgrade-invitee", "downgrade@example.com")
	invitations := repository.NewWorkspaceInvitationRepository(database)
	now := time.Now()
	createInvitation(t, invitations, owner, "downgrade-invitation", invitee.Email, "downgrade-secret", now)
	if _, err := database.Exec(`UPDATE subscriptions SET plan_id = $1 WHERE workspace_id = $2`, model.SubscriptionPlanPro, owner.Workspace.ID); err != nil {
		t.Fatalf("set Pro plan: %v", err)
	}

	tx, err := database.Beginx()
	if err != nil {
		t.Fatalf("begin downgrade: %v", err)
	}
	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, owner.Workspace.ID); err != nil {
		t.Fatalf("lock workspace: %v", err)
	}
	if _, err := tx.Exec(`UPDATE subscriptions SET plan_id = $1 WHERE workspace_id = $2`, model.SubscriptionPlanFree, owner.Workspace.ID); err != nil {
		t.Fatalf("stage downgrade: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := invitations.Accept(invitationHash("downgrade-secret"), invitee.ID, invitee.Email, "downgrade-member", "downgrade-audit", time.Now())
		result <- err
	}()
	time.Sleep(20 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit downgrade: %v", err)
	}
	if err := <-result; !errors.Is(err, repository.ErrSeatLimit) {
		t.Fatalf("Accept() after locked downgrade error = %v, want ErrSeatLimit", err)
	}
}

func TestInvitationAcceptanceValidatesEmailAndExpiry(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "validation-owner", Name: "Owner"})
	invitee := createBareUser(t, database, "validation-invitee", "invitee@example.com")
	workspaceInvitationRepository := repository.NewWorkspaceInvitationRepository(database)
	now := time.Now()

	createInvitation(t, workspaceInvitationRepository, owner, "email-invitation", invitee.Email, "email-secret", now)
	if _, err := workspaceInvitationRepository.Accept(invitationHash("email-secret"), invitee.ID, "someone-else@example.com", "wrong-email-member", "wrong-email-audit", now); !errors.Is(err, repository.ErrWorkspaceInvitationEmail) {
		t.Fatalf("wrong email error = %v, want ErrWorkspaceInvitationEmail", err)
	}

	expired := &model.WorkspaceInvitation{
		ID:          "expired-invitation",
		WorkspaceID: owner.Workspace.ID,
		Email:       "expired@example.com",
		Role:        model.WorkspaceRoleMember,
		TokenHash:   invitationHash("expired-secret"),
		InvitedBy:   owner.User.ID,
		ExpiresAt:   now.Add(-time.Minute),
		CreatedAt:   now.Add(-time.Hour),
	}
	if err := workspaceInvitationRepository.Create(expired, testAudit(owner.Workspace.ID, owner.User.ID, "expired-invitation")); err != nil {
		t.Fatalf("create expired invitation: %v", err)
	}
	if _, err := workspaceInvitationRepository.Accept(invitationHash("expired-secret"), invitee.ID, expired.Email, "expired-member", "expired-audit", now); !errors.Is(err, repository.ErrWorkspaceInvitationExpired) {
		t.Fatalf("expired invitation error = %v, want ErrWorkspaceInvitationExpired", err)
	}
}

func TestInvitationCannotBeAcceptedAndRevokedConcurrently(t *testing.T) {
	database := testutil.NewDatabase(t)
	owner := testutil.SeedUser(t, database, testutil.UserOptions{ID: "revoke-owner", Name: "Owner"})
	invitee := createBareUser(t, database, "revoke-invitee", "revoke@example.com")
	workspaceInvitationRepository := repository.NewWorkspaceInvitationRepository(database)
	createInvitation(t, workspaceInvitationRepository, owner, "revoke-race-invitation", invitee.Email, "revoke-race-secret", time.Now())
	if _, err := database.Exec(`UPDATE subscriptions SET plan_id = $1 WHERE workspace_id = $2`, model.SubscriptionPlanPro, owner.Workspace.ID); err != nil {
		t.Fatalf("set seat plan: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := workspaceInvitationRepository.Accept(invitationHash("revoke-race-secret"), invitee.ID, invitee.Email, "revoke-race-member", "revoke-race-accept-audit", time.Now())
		results <- err
	}()
	go func() {
		<-start
		results <- workspaceInvitationRepository.Revoke(owner.Workspace.ID, "revoke-race-invitation", time.Now(), testAudit(owner.Workspace.ID, owner.User.ID, "revoke-race"))
	}()
	close(start)

	successes := 0
	for range 2 {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, repository.ErrWorkspaceInvitationRevoked) && !errors.Is(err, repository.ErrWorkspaceInvitationNotFound) {
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent actions = %d, want 1", successes)
	}

	var acceptedAt, revokedAt *time.Time
	if err := database.QueryRowx(`SELECT accepted_at, revoked_at FROM workspace_invitations WHERE id = $1`, "revoke-race-invitation").Scan(&acceptedAt, &revokedAt); err != nil {
		t.Fatalf("load invitation state: %v", err)
	}
	if acceptedAt != nil && revokedAt != nil {
		t.Fatal("invitation is both accepted and revoked")
	}
}

func createBareUser(t *testing.T, database *sqlx.DB, id, email string) *model.User {
	t.Helper()
	now := time.Now()
	user := &model.User{ID: id, Email: email, Name: id, CreatedAt: now}
	if err := repository.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("create user %s: %v", id, err)
	}
	return user
}

func insertMembership(t *testing.T, database *sqlx.DB, id, workspaceID, userID, role string) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, workspaceID, userID, role, time.Now()); err != nil {
		t.Fatalf("insert member: %v", err)
	}
}

func createInvitation(t *testing.T, workspaceInvitationRepository repository.WorkspaceInvitationRepository, owner testutil.UserFixture, id, email, token string, now time.Time) {
	t.Helper()
	invitation := &model.WorkspaceInvitation{
		ID:          id,
		WorkspaceID: owner.Workspace.ID,
		Email:       email,
		Role:        model.WorkspaceRoleMember,
		TokenHash:   invitationHash(token),
		InvitedBy:   owner.User.ID,
		ExpiresAt:   now.Add(time.Hour),
		CreatedAt:   now,
	}
	if err := workspaceInvitationRepository.Create(invitation, testAudit(owner.Workspace.ID, owner.User.ID, "workspace.invitation.sent."+id)); err != nil {
		t.Fatalf("create invitation %s: %v", id, err)
	}
}

func invitationHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func testAudit(workspaceID, actorID, id string) *model.WorkspaceAuditEvent {
	return &model.WorkspaceAuditEvent{
		ID:          "audit-" + id,
		WorkspaceID: workspaceID,
		ActorUserID: &actorID,
		Action:      id,
		TargetType:  "test",
		TargetID:    id,
		Metadata:    "{}",
		CreatedAt:   time.Now(),
	}
}
