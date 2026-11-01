package repository_test

import (
	"testing"

	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestNewsletterRepositorySubscribesIdempotently(t *testing.T) {
	database := testutil.NewDatabase(t)
	repository := repository.NewNewsletterRepository(database)

	for range 2 {
		if err := repository.Subscribe("reader@example.com"); err != nil {
			t.Fatalf("Subscribe() error = %v", err)
		}
	}

	var count int
	if err := database.Get(&count, `SELECT COUNT(*) FROM newsletter_subscribers WHERE email = $1`, "reader@example.com"); err != nil {
		t.Fatalf("count subscribers: %v", err)
	}
	if count != 1 {
		t.Fatalf("subscriber count = %d, want 1", count)
	}
}
