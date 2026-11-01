package api

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"{{.ModulePath}}/internal/model"
)

// User is the public JSON shape of an account. It never carries the password
// hash or any token.
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	PasswordSet bool      `json:"password_set"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewUser converts a user model into its public JSON shape.
func NewUser(user *model.User) User {
	return User{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		PasswordSet: user.HasPassword(),
		CreatedAt:   user.CreatedAt.UTC(),
	}
}

// SessionToken is the answer of every sign-in endpoint. The token is the
// bearer credential for the Authorization header.
type SessionToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// CookieSession is the answer of a sign-in with ?session=cookie. The token
// is only in the HttpOnly session cookie, never in the body.
type CookieSession struct {
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// UserResponse wraps a user for endpoints that answer {"user": ...}.
type UserResponse struct {
	User User `json:"user"`
}

// utcTime returns a copy of value in UTC, or nil.
func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.UTC()
	return &converted
}

// FieldMessage turns a validation error into a sentence for the fields map.
func FieldMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "This value is invalid."
	}
	first, size := utf8.DecodeRuneInString(message)
	message = string(unicode.ToUpper(first)) + message[size:]
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message
}
