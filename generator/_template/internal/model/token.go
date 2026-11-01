package model

import (
	"time"
)

type Token struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Type      string    `db:"type"` // "email_verify" or "password_reset"
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

const (
	TokenTypeEmailVerify   = "email_verify"
	TokenTypePasswordReset = "password_reset"
	TokenTypeEmailChange   = "email_change"
	TokenTypeMagicLink     = "magic_link"
)

func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *Token) IsValid() bool {
	return !t.IsExpired()
}
