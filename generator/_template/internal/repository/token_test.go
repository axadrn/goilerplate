package repository_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestTokenCanBeConsumedOnlyOnce(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{Verified: true})
	tokenRepository := repository.NewTokenRepository(database)
	token := &model.Token{
		UserID:    fixture.User.ID,
		Type:      model.TokenTypeMagicLink,
		TokenHash: "single-use-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := tokenRepository.Create(token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if token.ID == "" || token.CreatedAt.IsZero() {
		t.Fatalf("Create() did not populate token metadata: %#v", token)
	}

	consumed, err := tokenRepository.ConsumeToken(token.TokenHash)
	if err != nil {
		t.Fatalf("first ConsumeToken() error = %v", err)
	}
	if consumed.UserID != fixture.User.ID {
		t.Fatalf("consumed token = %#v", consumed)
	}
	if _, err := tokenRepository.ConsumeToken(token.TokenHash); !errors.Is(err, repository.ErrTokenNotFound) {
		t.Fatalf("second ConsumeToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenConsumptionIsAtomic(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{Verified: true})
	tokenRepository := repository.NewTokenRepository(database)
	token := &model.Token{
		UserID:    fixture.User.ID,
		Type:      model.TokenTypeMagicLink,
		TokenHash: "concurrent-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := tokenRepository.Create(token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	const attempts = 8
	results := make(chan error, attempts)
	var waitGroup sync.WaitGroup
	for range attempts {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := tokenRepository.ConsumeToken(token.TokenHash)
			results <- err
		}()
	}
	waitGroup.Wait()
	close(results)

	var successes int
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, repository.ErrTokenNotFound) {
			t.Fatalf("ConsumeToken() error = %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful consumptions = %d, want 1", successes)
	}
}

func TestExpiredTokenCannotBeConsumed(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{Verified: true})
	tokenRepository := repository.NewTokenRepository(database)
	token := &model.Token{
		UserID:    fixture.User.ID,
		Type:      model.TokenTypeEmailChange,
		TokenHash: "expired-token",
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	if err := tokenRepository.Create(token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := tokenRepository.ConsumeToken(token.TokenHash); !errors.Is(err, repository.ErrTokenNotFound) {
		t.Fatalf("ConsumeToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestDeleteByUserAndTypePreservesOtherTokens(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{Verified: true})
	tokenRepository := repository.NewTokenRepository(database)
	for _, tokenType := range []string{model.TokenTypeMagicLink, model.TokenTypeEmailChange} {
		token := &model.Token{
			UserID:    fixture.User.ID,
			Type:      tokenType,
			TokenHash: tokenType + "-token",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := tokenRepository.Create(token); err != nil {
			t.Fatalf("Create(%s) error = %v", tokenType, err)
		}
	}

	if err := tokenRepository.DeleteByUserAndType(fixture.User.ID, model.TokenTypeMagicLink); err != nil {
		t.Fatalf("DeleteByUserAndType() error = %v", err)
	}
	if _, err := tokenRepository.ConsumeToken(model.TokenTypeMagicLink + "-token"); !errors.Is(err, repository.ErrTokenNotFound) {
		t.Fatalf("deleted token error = %v, want ErrTokenNotFound", err)
	}
	if _, err := tokenRepository.ConsumeToken(model.TokenTypeEmailChange + "-token"); err != nil {
		t.Fatalf("unrelated token was deleted: %v", err)
	}
}
