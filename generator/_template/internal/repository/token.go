package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token has expired")
	ErrTokenUsed     = errors.New("token has already been used")
)

type TokenRepository interface {
	Create(token *model.Token) error
	ConsumeToken(tokenHash string) (*model.Token, error)
	DeleteByUserAndType(userID, tokenType string) error
	DeleteExpired(now time.Time) error
}

type tokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(token *model.Token) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO tokens (id, user_id, type, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		token.ID,
		token.UserID,
		token.Type,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return err
}

// ConsumeToken atomically deletes and returns a valid single-use token.
func (r *tokenRepository) ConsumeToken(tokenHash string) (*model.Token, error) {
	var t model.Token
	now := time.Now()

	query := `
		DELETE FROM tokens
		WHERE token_hash = $1
		AND expires_at > $2
		RETURNING *
	`

	err := r.db.Get(&t, query, tokenHash, now)
	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (r *tokenRepository) DeleteByUserAndType(userID, tokenType string) error {
	query := `DELETE FROM tokens WHERE user_id = $1 AND type = $2`
	_, err := r.db.Exec(query, userID, tokenType)
	return err
}

func (r *tokenRepository) DeleteExpired(now time.Time) error {
	_, err := r.db.Exec(`DELETE FROM tokens WHERE expires_at <= $1`, now)
	return err
}
