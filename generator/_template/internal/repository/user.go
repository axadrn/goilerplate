package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateEmail = errors.New("email already exists")
)

type UserRepository interface {
	Create(user *model.User) error
	ByID(id string) (*model.User, error)
	ByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	UpdateName(userID, name string, updatedAt time.Time) error
	Delete(id string) error
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = user.CreatedAt
	}
	query := `INSERT INTO users (id, email, password_hash, email_verified_at, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(query, user.ID, user.Email, user.PasswordHash, user.EmailVerifiedAt, user.Name, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation (works for both SQLite and PostgreSQL)
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "duplicate key value") {
			return ErrDuplicateEmail
		}
		return err
	}

	return nil
}

func (r *userRepository) ByID(id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.Get(user, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return user, err
}

func (r *userRepository) ByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.Get(user, query, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return user, err
}

func (r *userRepository) Update(user *model.User) error {
	query := `UPDATE users SET email = $1, password_hash = $2, pending_email = $3, email_verified_at = $4, name = $5, updated_at = $6 WHERE id = $7`

	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, user.Email, user.PasswordHash, user.PendingEmail, user.EmailVerifiedAt, user.Name, user.UpdatedAt, user.ID)
	return err
}

func (r *userRepository) UpdateName(userID, name string, updatedAt time.Time) error {
	result, err := r.db.Exec(`UPDATE users SET name = $1, updated_at = $2 WHERE id = $3`, name, updatedAt, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
