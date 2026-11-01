package testutil

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

type UserOptions struct {
	ID       string
	Email    string
	Name     string
	Password string
	Verified bool
}

type UserFixture struct {
	User         *model.User
	Subscription *model.Subscription
}

func SeedUser(t testing.TB, database *sqlx.DB, options UserOptions) UserFixture {
	t.Helper()
	now := time.Now()
	if options.ID == "" {
		options.ID = "user-1"
	}
	if options.Email == "" {
		options.Email = options.ID + "@example.com"
	}

	var passwordHash *string
	if options.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(options.Password), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("hash fixture password: %v", err)
		}
		value := string(hash)
		passwordHash = &value
	}
	var verifiedAt *time.Time
	if options.Verified {
		verifiedAt = &now
	}

	user := &model.User{
		ID:              options.ID,
		Email:           options.Email,
		PasswordHash:    passwordHash,
		EmailVerifiedAt: verifiedAt,
		Name:            options.Name,
		CreatedAt:       now,
	}
	if err := repository.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("create fixture user: %v", err)
	}

	subscription := &model.Subscription{
		ID:        "subscription-" + options.ID,
		UserID:    options.ID,
		PlanID:    model.SubscriptionPlanFree,
		Status:    model.SubscriptionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repository.NewSubscriptionRepository(database).Create(subscription); err != nil {
		t.Fatalf("create fixture subscription: %v", err)
	}

	return UserFixture{User: user, Subscription: subscription}
}

type MemoryStorage struct {
	mu    sync.Mutex
	files map[string][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{files: make(map[string][]byte)}
}

func (s *MemoryStorage) Save(path string, file io.Reader) error {
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = data
	return nil
}

func (s *MemoryStorage) Delete(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, path)
	return nil
}

// Has reports whether a file is stored at path.
func (s *MemoryStorage) Has(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.files[path]
	return exists
}

func (s *MemoryStorage) URL(path string) string {
	return "https://storage.test/" + path
}
