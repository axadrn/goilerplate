package model

import (
	"time"
)

type User struct {
	ID              string     `db:"id"`
	Email           string     `db:"email"`
	PasswordHash    *string    `db:"password_hash"` // Nullable for passwordless users
	PendingEmail    *string    `db:"pending_email"`
	EmailVerifiedAt *time.Time `db:"email_verified_at"`
	Name            string     `db:"name"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`

	// Computed fields (not in database)
	AvatarURL   string `db:"-"`
	PasswordSet bool   `db:"-"`
}

func (u *User) HasPassword() bool {
	return u.PasswordSet || (u.PasswordHash != nil && *u.PasswordHash != "")
}
