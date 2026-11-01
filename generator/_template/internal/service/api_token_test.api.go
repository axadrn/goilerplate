package service

import (
	"errors"
	"testing"
	"time"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

// memoryAPITokens is an in-memory APITokenRepository that counts writes.
type memoryAPITokens struct {
	byHash  map[string]*model.APIToken
	touches int
}

func (m *memoryAPITokens) Create(token *model.APIToken) error {
	copied := *token
	m.byHash[token.TokenHash] = &copied
	return nil
}

func (m *memoryAPITokens) ByTokenHash(tokenHash string) (*model.APIToken, error) {
	token, ok := m.byHash[tokenHash]
	if !ok {
		return nil, repository.ErrAPITokenNotFound
	}
	copied := *token
	return &copied, nil
}

func (m *memoryAPITokens) ListByUserID(string) ([]model.APIToken, error) { return nil, nil }

func (m *memoryAPITokens) CountByUserID(string) (int, error) { return len(m.byHash), nil }

func (m *memoryAPITokens) TouchLastUsed(id string, lastUsedAt time.Time) error {
	m.touches++
	for _, token := range m.byHash {
		if token.ID == id {
			used := lastUsedAt
			token.LastUsedAt = &used
		}
	}
	return nil
}

func (m *memoryAPITokens) DeleteByIDForUser(string, string) error { return nil }

func (m *memoryAPITokens) DeleteByUserID(userID string) error {
	for hash, token := range m.byHash {
		if token.UserID == userID {
			delete(m.byHash, hash)
		}
	}
	return nil
}

// recordingSessions records which sessions AuthService asked to delete.
type recordingSessions struct {
	repository.SessionRepository
	deletedAllFor string
	keptSessionID string
}

func (r *recordingSessions) DeleteAllForUser(userID string) error {
	r.deletedAllFor = userID
	return nil
}

func (r *recordingSessions) DeleteOthersForUser(userID, currentSessionID string) error {
	r.deletedAllFor = userID
	r.keptSessionID = currentSessionID
	return nil
}

func TestRevokeAccessRevokesAPITokens(t *testing.T) {
	for _, test := range []struct {
		name   string
		revoke func(*AuthService) error
		kept   string
	}{
		{name: "all", revoke: func(auth *AuthService) error { return auth.RevokeAllAccess("user") }},
		{name: "others", revoke: func(auth *AuthService) error { return auth.RevokeOtherAccess("user", "current") }, kept: "current"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &memoryAPITokens{byHash: map[string]*model.APIToken{
				"user-1":  {ID: "user-1", UserID: "user"},
				"user-2":  {ID: "user-2", UserID: "user"},
				"other-1": {ID: "other-1", UserID: "other"},
			}}
			sessions := &recordingSessions{}
			auth := &AuthService{sessionRepository: sessions}

			err := test.revoke(auth)
			if !errors.Is(err, errAPITokensNotConfigured) {
				t.Fatalf("revoke without api tokens error = %v, want %v", err, errAPITokensNotConfigured)
			}

			auth.UseAPITokens(&APITokenService{tokens: store, now: time.Now})
			err = test.revoke(auth)
			if err != nil {
				t.Fatal(err)
			}
			if sessions.deletedAllFor != "user" || sessions.keptSessionID != test.kept {
				t.Fatalf("sessions deleted for %q keeping %q", sessions.deletedAllFor, sessions.keptSessionID)
			}
			if len(store.byHash) != 1 || store.byHash["other-1"] == nil {
				t.Fatalf("tokens left = %#v", store.byHash)
			}
		})
	}
}

func TestAPITokenAuthenticateRecordsUseAtMostOncePerMinute(t *testing.T) {
	store := &memoryAPITokens{byHash: map[string]*model.APIToken{}}
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	service := &APITokenService{tokens: store, now: func() time.Time { return now }}
	expiresAt := now.Add(2 * time.Minute)
	raw := APITokenMarker + "abcdefghijklmnopqrstuvwxyz234567abcdefghijklmnopqrst"
	err := store.Create(&model.APIToken{ID: "token", UserID: "user", TokenHash: hashToken(raw), Prefix: "abcdefgh", Scopes: "read", ExpiresAt: &expiresAt, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range []struct {
		advance time.Duration
		touches int
	}{
		{advance: 0, touches: 1},
		{advance: 30 * time.Second, touches: 1},
		{advance: 30 * time.Second, touches: 2},
		{advance: 59 * time.Second, touches: 2},
	} {
		now = now.Add(step.advance)
		token, err := service.Authenticate(raw)
		if err != nil {
			t.Fatalf("Authenticate() at %s error = %v", now, err)
		}
		if token.UserID != "user" || store.touches != step.touches {
			t.Fatalf("at %s: user %q, touches %d, want %d", now, token.UserID, store.touches, step.touches)
		}
	}

	now = expiresAt
	_, err = service.Authenticate(raw)
	if !errors.Is(err, repository.ErrAPITokenNotFound) {
		t.Fatalf("expired token error = %v, want ErrAPITokenNotFound", err)
	}
	for _, bogus := range []string{"", "pat_", "session-token", APITokenMarker + "unknown"} {
		_, err := service.Authenticate(bogus)
		if err == nil {
			t.Fatalf("Authenticate(%q) succeeded", bogus)
		}
	}
}

func TestNormalizeAPITokenScopes(t *testing.T) {
	for _, test := range []struct {
		requested []string
		want      string
		fails     bool
	}{
		{requested: nil, want: "read"},
		{requested: []string{"read"}, want: "read"},
		{requested: []string{"write"}, want: "read,write"},
		{requested: []string{"write", "read", "read"}, want: "read,write"},
		{requested: []string{"read", "admin"}, fails: true},
		{requested: []string{""}, fails: true},
	} {
		got, err := normalizeAPITokenScopes(test.requested)
		if test.fails {
			if !errors.Is(err, ErrAPITokenScope) {
				t.Fatalf("%v: error = %v, want ErrAPITokenScope", test.requested, err)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Fatalf("%v = %q, %v, want %q", test.requested, got, err, test.want)
		}
	}
}
