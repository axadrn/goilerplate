package repository

import "github.com/jmoiron/sqlx"

type NewsletterRepository interface {
	Subscribe(email string) error
}

type newsletterRepository struct {
	db *sqlx.DB
}

func NewNewsletterRepository(db *sqlx.DB) NewsletterRepository {
	return &newsletterRepository{db: db}
}

func (r *newsletterRepository) Subscribe(email string) error {
	const query = `
		INSERT INTO newsletter_subscribers (email)
		VALUES ($1)
		ON CONFLICT (email) DO NOTHING
	`
	_, err := r.db.Exec(query, email)
	return err
}
