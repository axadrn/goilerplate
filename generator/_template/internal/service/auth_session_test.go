package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestDatabaseSessionLifecycle(t *testing.T) {
	database := sessionTestDatabase(t)
	userID := "user-1"
	seedSessionTestUser(t, database, userID)

	sessionRepository := repository.NewSessionRepository(database)
	authService := NewAuthService(
		nil,
		nil,
		sessionRepository,
		nil,
		nil,
		true,
		7*24*time.Hour,
		24*time.Hour,
		time.Hour,
		24*time.Hour,
		10*time.Minute,
	)

	request := httptest.NewRequest("GET", "https://example.com/auth", nil)
	request.Header.Set("User-Agent", "Session test browser")
	recorder := httptest.NewRecorder()

	created, err := authService.StartSession(recorder, request, userID)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "session_token" || cookie.Value == "" {
		t.Fatalf("session cookie = %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie security attributes = %#v", cookie)
	}

	var storedToken string
	if err := database.Get(&storedToken, `SELECT token_hash FROM sessions WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("query stored token: %v", err)
	}
	if storedToken == cookie.Value || storedToken != hashToken(cookie.Value) {
		t.Fatal("database must contain only the session token hash")
	}

	authenticated, err := authService.AuthenticateSession(cookie.Value)
	if err != nil {
		t.Fatalf("AuthenticateSession() error = %v", err)
	}
	if authenticated.ID != created.ID || authenticated.UserID != userID {
		t.Fatalf("authenticated session = %#v, want id %s and user %s", authenticated, created.ID, userID)
	}

	if err := authService.RevokeSession(userID, created.ID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if _, err := authService.AuthenticateSession(cookie.Value); err == nil {
		t.Fatal("AuthenticateSession() succeeded after revocation")
	}
}

func TestStartSessionTruncatesUserAgent(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	request := httptest.NewRequest("GET", "http://example.com/auth", nil)
	request.Header.Set("User-Agent", strings.Repeat("a", 600))
	session, err := authService.StartSession(httptest.NewRecorder(), request, "user-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if len(session.UserAgent) != 512 {
		t.Fatalf("user agent length = %d, want 512", len(session.UserAgent))
	}
}

func TestStartSessionRotatesExistingBrowserSession(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest("GET", "http://example.com/auth", nil)
	if _, err := authService.StartSession(firstRecorder, firstRequest, "user-1"); err != nil {
		t.Fatalf("start first session: %v", err)
	}
	firstCookie := firstRecorder.Result().Cookies()[0]

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest("GET", "http://example.com/auth", nil)
	secondRequest.AddCookie(firstCookie)
	if _, err := authService.StartSession(secondRecorder, secondRequest, "user-1"); err != nil {
		t.Fatalf("rotate session: %v", err)
	}
	secondToken := secondRecorder.Result().Cookies()[0].Value

	if _, err := authService.AuthenticateSession(firstCookie.Value); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("old session error = %v, want ErrSessionNotFound", err)
	}
	if _, err := authService.AuthenticateSession(secondToken); err != nil {
		t.Fatalf("new session error = %v", err)
	}
}

func TestStartFreshSessionRevokesEveryExistingSession(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	sessionRepository := repository.NewSessionRepository(database)
	authService := NewAuthService(
		nil, nil, sessionRepository, nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	firstRecorder := httptest.NewRecorder()
	if _, err := authService.StartSession(firstRecorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "user-1"); err != nil {
		t.Fatalf("start first session: %v", err)
	}
	secondRecorder := httptest.NewRecorder()
	if _, err := authService.StartSession(secondRecorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "user-1"); err != nil {
		t.Fatalf("start second session: %v", err)
	}

	freshRecorder := httptest.NewRecorder()
	fresh, err := authService.StartFreshSession(freshRecorder, httptest.NewRequest("GET", "http://example.com/recover", nil), "user-1")
	if err != nil {
		t.Fatalf("StartFreshSession() error = %v", err)
	}

	for _, oldToken := range []string{
		firstRecorder.Result().Cookies()[0].Value,
		secondRecorder.Result().Cookies()[0].Value,
	} {
		if _, err := authService.AuthenticateSession(oldToken); !errors.Is(err, repository.ErrSessionNotFound) {
			t.Fatalf("old session error = %v, want ErrSessionNotFound", err)
		}
	}
	freshToken := freshRecorder.Result().Cookies()[0].Value
	if _, err := authService.AuthenticateSession(freshToken); err != nil {
		t.Fatalf("fresh session error = %v", err)
	}
	sessions, err := sessionRepository.ListByUserID("user-1", time.Now())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != fresh.ID {
		t.Fatalf("sessions = %#v, want only fresh session %s", sessions, fresh.ID)
	}
}

func TestCreateSessionReturnsTokenWithoutCookie(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	existingRecorder := httptest.NewRecorder()
	_, err := authService.StartSession(existingRecorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "user-1")
	if err != nil {
		t.Fatalf("start browser session: %v", err)
	}
	existingCookie := existingRecorder.Result().Cookies()[0]

	request := httptest.NewRequest("POST", "http://example.com/api/v1/auth/password", nil)
	request.AddCookie(existingCookie)
	created, token, err := authService.CreateSession(request, "user-1")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if token == "" || created.TokenHash != hashToken(token) {
		t.Fatalf("CreateSession() token = %q, hash = %q", token, created.TokenHash)
	}

	authenticated, err := authService.AuthenticateSession(token)
	if err != nil {
		t.Fatalf("AuthenticateSession() error = %v", err)
	}
	if authenticated.ID != created.ID {
		t.Fatalf("authenticated session = %s, want %s", authenticated.ID, created.ID)
	}
	_, err = authService.AuthenticateSession(existingCookie.Value)
	if err != nil {
		t.Fatalf("CreateSession() must not replace the browser session: %v", err)
	}
}

func TestCreateFreshSessionRevokesEveryExistingSession(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	sessionRepository := repository.NewSessionRepository(database)
	authService := NewAuthService(
		nil, nil, sessionRepository, nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	request := httptest.NewRequest("POST", "http://example.com/api/v1/auth/forgot-password/verify", nil)
	_, oldToken, err := authService.CreateSession(request, "user-1")
	if err != nil {
		t.Fatalf("create old session: %v", err)
	}
	fresh, freshToken, err := authService.CreateFreshSession(request, "user-1")
	if err != nil {
		t.Fatalf("CreateFreshSession() error = %v", err)
	}

	_, err = authService.AuthenticateSession(oldToken)
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("old session error = %v, want ErrSessionNotFound", err)
	}
	authenticated, err := authService.AuthenticateSession(freshToken)
	if err != nil {
		t.Fatalf("fresh session error = %v", err)
	}
	if authenticated.ID != fresh.ID {
		t.Fatalf("authenticated session = %s, want %s", authenticated.ID, fresh.ID)
	}
	sessions, err := sessionRepository.ListByUserID("user-1", time.Now())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}
}

func TestRevokeSessionByTokenDeletesOnlyThatSession(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	request := httptest.NewRequest("POST", "http://example.com/api/v1/auth/password", nil)
	_, revokedToken, err := authService.CreateSession(request, "user-1")
	if err != nil {
		t.Fatalf("create revoked session: %v", err)
	}
	_, keptToken, err := authService.CreateSession(request, "user-1")
	if err != nil {
		t.Fatalf("create kept session: %v", err)
	}

	err = authService.RevokeSessionByToken(revokedToken)
	if err != nil {
		t.Fatalf("RevokeSessionByToken() error = %v", err)
	}
	_, err = authService.AuthenticateSession(revokedToken)
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("revoked session error = %v, want ErrSessionNotFound", err)
	}
	_, err = authService.AuthenticateSession(keptToken)
	if err != nil {
		t.Fatalf("other session was revoked: %v", err)
	}
	err = authService.RevokeSessionByToken("")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("empty token error = %v, want ErrSessionNotFound", err)
	}
}

func TestRevokeOtherSessionsKeepsCurrentSession(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	currentRecorder := httptest.NewRecorder()
	current, err := authService.StartSession(currentRecorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "user-1")
	if err != nil {
		t.Fatalf("start current session: %v", err)
	}
	otherRecorder := httptest.NewRecorder()
	if _, err := authService.StartSession(otherRecorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "user-1"); err != nil {
		t.Fatalf("start other session: %v", err)
	}

	if err := authService.RevokeOtherSessions("user-1", current.ID); err != nil {
		t.Fatalf("RevokeOtherSessions() error = %v", err)
	}
	if _, err := authService.AuthenticateSession(currentRecorder.Result().Cookies()[0].Value); err != nil {
		t.Fatalf("current session was revoked: %v", err)
	}
	if _, err := authService.AuthenticateSession(otherRecorder.Result().Cookies()[0].Value); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("other session error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionRevocationIsScopedToUser(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	seedSessionTestUser(t, database, "user-2")
	authService := NewAuthService(
		nil, nil, repository.NewSessionRepository(database), nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	request := httptest.NewRequest("GET", "http://example.com/auth", nil)
	first, err := authService.StartSession(httptest.NewRecorder(), request, "user-1")
	if err != nil {
		t.Fatalf("start first session: %v", err)
	}
	secondRecorder := httptest.NewRecorder()
	second, err := authService.StartSession(secondRecorder, request, "user-2")
	if err != nil {
		t.Fatalf("start second session: %v", err)
	}
	secondToken := secondRecorder.Result().Cookies()[0].Value

	if err := authService.RevokeSession("user-1", second.ID); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("cross-user RevokeSession() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := authService.AuthenticateSession(secondToken); err != nil {
		t.Fatalf("second user's session was revoked: %v", err)
	}
	if err := authService.RevokeAllSessions("user-1"); err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}
	if _, err := authService.sessionRepository.ByTokenHash(first.TokenHash); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("first user's session still exists: %v", err)
	}
	if _, err := authService.AuthenticateSession(secondToken); err != nil {
		t.Fatalf("second user's session was affected by revoke all: %v", err)
	}
}

func TestExpiredSessionIsRejectedAndRemoved(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "user-1")
	sessionRepository := repository.NewSessionRepository(database)
	authService := NewAuthService(
		nil, nil, sessionRepository, nil, nil,
		false, -time.Minute, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "http://example.com/auth", nil)
	created, err := authService.StartSession(recorder, request, "user-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	token := recorder.Result().Cookies()[0].Value
	if _, err := authService.AuthenticateSession(token); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("AuthenticateSession() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := sessionRepository.ByTokenHash(created.TokenHash); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expired session still exists: %v", err)
	}
}

func TestAuthenticationCleansExpiredSessionsForDormantUsers(t *testing.T) {
	database := sessionTestDatabase(t)
	seedSessionTestUser(t, database, "dormant-user")
	seedSessionTestUser(t, database, "active-user")
	sessionRepository := repository.NewSessionRepository(database)
	now := time.Now()
	expired := &model.Session{
		ID:         "expired-session",
		UserID:     "dormant-user",
		TokenHash:  hashToken("expired-token"),
		ExpiresAt:  now.Add(-time.Hour),
		LastSeenAt: now.Add(-2 * time.Hour),
		CreatedAt:  now.Add(-2 * time.Hour),
	}
	if err := sessionRepository.Create(expired); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	authService := NewAuthService(
		nil, nil, sessionRepository, nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)
	recorder := httptest.NewRecorder()
	if _, err := authService.StartSession(recorder, httptest.NewRequest("GET", "http://example.com/auth", nil), "active-user"); err != nil {
		t.Fatalf("start active session: %v", err)
	}
	if _, err := authService.AuthenticateSession(recorder.Result().Cookies()[0].Value); err != nil {
		t.Fatalf("authenticate active session: %v", err)
	}
	if _, err := sessionRepository.ByTokenHash(expired.TokenHash); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expired dormant session still exists: %v", err)
	}
}

func TestActivityUpdateFailureDoesNotInvalidateSession(t *testing.T) {
	now := time.Now()
	repositoryWithFailure := &activityFailureSessionRepository{
		session: &model.Session{
			ID:         "session-1",
			UserID:     "user-1",
			TokenHash:  hashToken("valid-token"),
			ExpiresAt:  now.Add(time.Hour),
			LastSeenAt: now.Add(-10 * time.Minute),
			CreatedAt:  now.Add(-time.Hour),
		},
	}
	authService := NewAuthService(
		nil, nil, repositoryWithFailure, nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)

	session, err := authService.AuthenticateSession("valid-token")
	if err != nil {
		t.Fatalf("AuthenticateSession() error = %v", err)
	}
	if session.ID != "session-1" {
		t.Fatalf("session ID = %q, want session-1", session.ID)
	}
}

type activityFailureSessionRepository struct {
	repository.SessionRepository
	session *model.Session
}

func (r *activityFailureSessionRepository) ByTokenHash(string) (*model.Session, error) {
	return r.session, nil
}

func (r *activityFailureSessionRepository) UpdateLastSeen(string, time.Time) error {
	return errors.New("temporary write failure")
}

func (r *activityFailureSessionRepository) DeleteExpired(time.Time) error {
	return nil
}

func sessionTestDatabase(t *testing.T) *sqlx.DB {
	t.Helper()
	return testutil.NewDatabase(t)
}

func seedSessionTestUser(t *testing.T, database *sqlx.DB, userID string) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO users (id, email, email_verified_at, created_at) VALUES ($1, $2, $3, $4)`,
		userID,
		userID+"@example.com",
		time.Now(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}
