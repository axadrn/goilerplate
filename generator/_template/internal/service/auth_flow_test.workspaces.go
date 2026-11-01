package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

const validTestPassword = "correct horse battery staple"

type authFlowEnvironment struct {
	database               *sqlx.DB
	auth                   *AuthService
	userRepository         repository.UserRepository
	subscriptionRepository repository.SubscriptionRepository
	workspaceService       *WorkspaceService
}

func newAuthFlowEnvironment(t *testing.T) authFlowEnvironment {
	t.Helper()
	database := testutil.NewDatabase(t)
	userRepository := repository.NewUserRepository(database)
	tokenRepository := repository.NewTokenRepository(database)
	sessionRepository := repository.NewSessionRepository(database)
	subscriptionRepository := repository.NewSubscriptionRepository(database)
	subscriptionService := NewSubscriptionService(subscriptionRepository, repository.NewAdminRepository(database))
	emailService := NewEmailService(testutil.NewMemoryMailer(), "http://localhost:8090", "Test App")
	workspaceService := NewWorkspaceService(
		repository.NewWorkspaceRepository(database),
		repository.NewWorkspaceMemberRepository(database),
		repository.NewWorkspaceInvitationRepository(database),
		repository.NewWorkspaceAuditRepository(database),
		userRepository,
		subscriptionService,
		emailService,
	)
	auth := NewAuthService(
		userRepository,
		tokenRepository,
		sessionRepository,
		workspaceService,
		emailService,
		false,
		7*24*time.Hour,
		24*time.Hour,
		time.Hour,
		24*time.Hour,
		10*time.Minute,
	)
	return authFlowEnvironment{
		database:               database,
		auth:                   auth,
		userRepository:         userRepository,
		subscriptionRepository: subscriptionRepository,
		workspaceService:       workspaceService,
	}
}

func TestOnboardingCreatesWorkspaceExceptForInvitedSignup(t *testing.T) {
	t.Run("normal signup", func(t *testing.T) {
		environment := newAuthFlowEnvironment(t)
		if err := environment.auth.SendMagicLink("normal@example.com"); err != nil {
			t.Fatalf("SendMagicLink() error = %v", err)
		}
		user, _ := environment.userRepository.ByEmail("normal@example.com")
		workspace, err := environment.auth.CompleteOnboarding(user.ID, "Normal User", true)
		if err != nil {
			t.Fatalf("CompleteOnboarding() error = %v", err)
		}
		if workspace == nil {
			t.Fatal("normal onboarding did not create a workspace")
		}
		members, err := environment.workspaceService.ListForUser(user.ID)
		if err != nil || len(members) != 1 || members[0].Role != model.WorkspaceRoleOwner {
			t.Fatalf("members = %#v, error = %v", members, err)
		}
		subscription, err := environment.subscriptionRepository.ByWorkspaceID(workspace.ID)
		if err != nil {
			t.Fatalf("workspace subscription lookup: %v", err)
		}
		if subscription.Provider != "" {
			t.Fatalf("free workspace provider = %q, want empty", subscription.Provider)
		}
	})

	t.Run("invited signup", func(t *testing.T) {
		environment := newAuthFlowEnvironment(t)
		if err := environment.auth.SendMagicLink("invited@example.com"); err != nil {
			t.Fatalf("SendMagicLink() error = %v", err)
		}
		user, _ := environment.userRepository.ByEmail("invited@example.com")
		workspace, err := environment.auth.CompleteOnboarding(user.ID, "Invited User", false)
		if err != nil {
			t.Fatalf("CompleteOnboarding() error = %v", err)
		}
		if workspace != nil {
			t.Fatalf("invited onboarding workspace = %#v, want nil", workspace)
		}
		members, err := environment.workspaceService.ListForUser(user.ID)
		if err != nil || len(members) != 0 {
			t.Fatalf("invited onboarding members = %#v, error = %v", members, err)
		}
	})

	t.Run("long valid name uses safe workspace fallback", func(t *testing.T) {
		environment := newAuthFlowEnvironment(t)
		if err := environment.auth.SendMagicLink("long-name@example.com"); err != nil {
			t.Fatalf("SendMagicLink() error = %v", err)
		}
		user, _ := environment.userRepository.ByEmail("long-name@example.com")
		name := strings.Repeat("A", 100)
		workspace, err := environment.auth.CompleteOnboarding(user.ID, name, true)
		if err != nil {
			t.Fatalf("CompleteOnboarding() error = %v", err)
		}
		if workspace.Name != "My Workspace" {
			t.Fatalf("workspace name = %q, want My Workspace", workspace.Name)
		}
		updated, err := environment.userRepository.ByID(user.ID)
		if err != nil || updated.Name != name {
			t.Fatalf("user name = %q, error = %v", updated.Name, err)
		}
	})

	t.Run("control characters are rejected before trimming", func(t *testing.T) {
		environment := newAuthFlowEnvironment(t)
		if err := environment.auth.SendMagicLink("control-name@example.com"); err != nil {
			t.Fatalf("SendMagicLink() error = %v", err)
		}
		user, _ := environment.userRepository.ByEmail("control-name@example.com")
		for _, name := range []string{"Jane\n", "\tJane", "Jane\r"} {
			if _, err := environment.auth.CompleteOnboarding(user.ID, name, true); err == nil {
				t.Fatalf("CompleteOnboarding(%q) error = nil", name)
			}
		}
		updated, err := environment.userRepository.ByID(user.ID)
		if err != nil || updated.Name != "" {
			t.Fatalf("user name = %q, error = %v", updated.Name, err)
		}
	})
}

func TestLoginValidatesCredentialsAndVerification(t *testing.T) {
	environment := newAuthFlowEnvironment(t)
	verified := testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:       "verified-user",
		Email:    "verified@example.com",
		Name:     "Verified User",
		Password: validTestPassword,
		Verified: true,
	})
	testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:       "unverified-user",
		Email:    "unverified@example.com",
		Password: validTestPassword,
	})
	testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:       "passwordless-user",
		Email:    "passwordless@example.com",
		Verified: true,
	})

	user, err := environment.auth.Login("  VERIFIED@EXAMPLE.COM ", validTestPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if user.ID != verified.User.ID {
		t.Fatalf("Login() user ID = %q, want %q", user.ID, verified.User.ID)
	}

	if _, err := environment.auth.Login(verified.User.Email, "wrong password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := environment.auth.Login("missing@example.com", validTestPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("missing user error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := environment.auth.Login("unverified@example.com", validTestPassword); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("unverified user error = %v, want ErrEmailNotVerified", err)
	}
	if _, err := environment.auth.Login("passwordless@example.com", validTestPassword); err == nil || !strings.Contains(err.Error(), "passwordless") {
		t.Fatalf("passwordless user error = %v", err)
	}
}

func TestPasswordLifecycle(t *testing.T) {
	environment := newAuthFlowEnvironment(t)
	fixture := testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:       "password-user",
		Email:    "password@example.com",
		Verified: true,
	})

	if err := environment.auth.SetPassword(fixture.User.ID, "short"); err == nil {
		t.Fatal("SetPassword() accepted a weak password")
	}
	if err := environment.auth.SetPassword(fixture.User.ID, validTestPassword); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	if _, err := environment.auth.Login(fixture.User.Email, validTestPassword); err != nil {
		t.Fatalf("Login() after SetPassword() error = %v", err)
	}
	if err := environment.auth.SetPassword(fixture.User.ID, validTestPassword); err == nil {
		t.Fatal("SetPassword() replaced an existing password")
	}
	if err := environment.auth.RemovePassword(fixture.User.ID); err != nil {
		t.Fatalf("RemovePassword() error = %v", err)
	}
	if _, err := environment.auth.Login(fixture.User.Email, validTestPassword); err == nil || !strings.Contains(err.Error(), "passwordless") {
		t.Fatalf("Login() after RemovePassword() error = %v", err)
	}
	if err := environment.auth.RemovePassword(fixture.User.ID); err == nil {
		t.Fatal("RemovePassword() removed an absent password")
	}
}

func TestMagicLinkSignupAndSingleUse(t *testing.T) {
	environment := newAuthFlowEnvironment(t)
	const email = "new-user@example.com"

	if err := environment.auth.SendMagicLink("  NEW-USER@EXAMPLE.COM "); err != nil {
		t.Fatalf("SendMagicLink() error = %v", err)
	}
	user, err := environment.userRepository.ByEmail(email)
	if err != nil {
		t.Fatalf("new user lookup: %v", err)
	}
	if user.HasPassword() || user.EmailVerifiedAt != nil {
		t.Fatalf("new magic-link user = %#v", user)
	}
	var subscriptions int
	if err := environment.database.Get(&subscriptions, `SELECT COUNT(*) FROM subscriptions`); err != nil || subscriptions != 0 {
		t.Fatalf("new user subscriptions = %d, error = %v, want none before onboarding", subscriptions, err)
	}

	const rawToken = "known-magic-link-token"
	if _, err := environment.database.Exec(`UPDATE tokens SET token_hash = $1 WHERE user_id = $2 AND type = $3`, hashToken(rawToken), user.ID, model.TokenTypeMagicLink); err != nil {
		t.Fatalf("set known magic-link token: %v", err)
	}
	authenticated, err := environment.auth.VerifyMagicLink(rawToken)
	if err != nil {
		t.Fatalf("VerifyMagicLink() error = %v", err)
	}
	if authenticated.EmailVerifiedAt == nil {
		t.Fatal("VerifyMagicLink() did not verify the email")
	}
	if _, err := environment.auth.VerifyMagicLink(rawToken); err == nil {
		t.Fatal("VerifyMagicLink() accepted a reused token")
	}
}

func TestSendMagicLinkReplacesPreviousToken(t *testing.T) {
	environment := newAuthFlowEnvironment(t)
	fixture := testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:       "magic-user",
		Email:    "magic@example.com",
		Verified: true,
	})

	if err := environment.auth.SendMagicLink(fixture.User.Email); err != nil {
		t.Fatalf("first SendMagicLink() error = %v", err)
	}
	var firstToken string
	if err := environment.database.Get(&firstToken, `SELECT token_hash FROM tokens WHERE user_id = $1 AND type = $2`, fixture.User.ID, model.TokenTypeMagicLink); err != nil {
		t.Fatalf("first token lookup: %v", err)
	}
	if err := environment.auth.SendMagicLink(fixture.User.Email); err != nil {
		t.Fatalf("second SendMagicLink() error = %v", err)
	}
	var tokens []string
	if err := environment.database.Select(&tokens, `SELECT token_hash FROM tokens WHERE user_id = $1 AND type = $2`, fixture.User.ID, model.TokenTypeMagicLink); err != nil {
		t.Fatalf("active token lookup: %v", err)
	}
	if len(tokens) != 1 || tokens[0] == firstToken {
		t.Fatalf("active tokens = %#v, want one rotated token", tokens)
	}
}

func TestOAuthProvisioningAndExistingUserVerification(t *testing.T) {
	environment := newAuthFlowEnvironment(t)

	created, err := environment.auth.AuthenticateOAuth("  OAUTH@EXAMPLE.COM ", "github")
	if err != nil {
		t.Fatalf("AuthenticateOAuth(new user) error = %v", err)
	}
	if created.Email != "oauth@example.com" || created.EmailVerifiedAt == nil {
		t.Fatalf("created OAuth user = %#v", created)
	}
	var subscriptions int
	if err := environment.database.Get(&subscriptions, `SELECT COUNT(*) FROM subscriptions`); err != nil || subscriptions != 0 {
		t.Fatalf("OAuth user subscriptions = %d, error = %v, want none before onboarding", subscriptions, err)
	}

	existing := testutil.SeedUser(t, environment.database, testutil.UserOptions{
		ID:    "existing-oauth-user",
		Email: "existing-oauth@example.com",
	})
	authenticated, err := environment.auth.AuthenticateOAuth(existing.User.Email, "google")
	if err != nil {
		t.Fatalf("AuthenticateOAuth(existing user) error = %v", err)
	}
	if authenticated.ID != existing.User.ID || authenticated.EmailVerifiedAt == nil {
		t.Fatalf("authenticated OAuth user = %#v", authenticated)
	}
}
