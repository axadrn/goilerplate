package repository

import (
	"database/sql"
	"errors"
	"time"

	"{{.ModulePath}}/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepository interface {
	Create(session *model.Session) error
	ReplaceAllForUser(session *model.Session) error
	ByTokenHash(tokenHash string) (*model.Session, error)
	ListByUserID(userID string, now time.Time) ([]model.Session, error)
	UpdateLastSeen(id string, lastSeenAt time.Time) error
	DeleteByTokenHash(tokenHash string) error
	DeleteByIDForUser(id, userID string) error
	DeleteOthersForUser(userID, keepSessionID string) error
	DeleteAllForUser(userID string) error
	DeleteExpired(now time.Time) error
}

type sessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(session *model.Session) error {
	_, err := r.db.Exec(
		insertSessionQuery,
		session.ID,
		session.UserID,
		session.ImpersonatorUserID,
		session.TokenHash,
		session.UserAgent,
		session.ExpiresAt,
		session.LastSeenAt,
		session.CreatedAt,
	)
	return err
}

const insertSessionQuery = `
		INSERT INTO sessions (
			id, user_id, impersonator_user_id, token_hash, user_agent, expires_at, last_seen_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

func (r *sessionRepository) ReplaceAllForUser(session *model.Session) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = $1 OR impersonator_user_id = $1`, session.UserID); err != nil {
		return err
	}
	if _, err := tx.Exec(
		insertSessionQuery,
		session.ID,
		session.UserID,
		session.ImpersonatorUserID,
		session.TokenHash,
		session.UserAgent,
		session.ExpiresAt,
		session.LastSeenAt,
		session.CreatedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sessionRepository) ByTokenHash(tokenHash string) (*model.Session, error) {
	session := &model.Session{}
	err := r.db.Get(session, `SELECT * FROM sessions WHERE token_hash = $1`, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return session, err
}

func (r *sessionRepository) ListByUserID(userID string, now time.Time) ([]model.Session, error) {
	if _, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = $1 AND expires_at <= $2`, userID, now); err != nil {
		return nil, err
	}

	sessions := []model.Session{}
	err := r.db.Select(&sessions, `
		SELECT * FROM sessions
		WHERE user_id = $1
		ORDER BY last_seen_at DESC, created_at DESC
	`, userID)
	return sessions, err
}

func (r *sessionRepository) UpdateLastSeen(id string, lastSeenAt time.Time) error {
	result, err := r.db.Exec(`UPDATE sessions SET last_seen_at = $1 WHERE id = $2`, lastSeenAt, id)
	if err != nil {
		return err
	}
	return requireSessionRow(result)
}

func (r *sessionRepository) DeleteByTokenHash(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *sessionRepository) DeleteByIDForUser(id, userID string) error {
	result, err := r.db.Exec(`DELETE FROM sessions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	return requireSessionRow(result)
}

func (r *sessionRepository) DeleteOthersForUser(userID, keepSessionID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE (user_id = $1 OR impersonator_user_id = $1) AND id <> $2`, userID, keepSessionID)
	return err
}

func (r *sessionRepository) DeleteAllForUser(userID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = $1 OR impersonator_user_id = $1`, userID)
	return err
}

func (r *sessionRepository) DeleteExpired(now time.Time) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= $1`, now)
	return err
}

func requireSessionRow(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}
