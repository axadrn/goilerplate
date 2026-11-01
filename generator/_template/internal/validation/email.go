package validation

import (
	"errors"
	"net/mail"
)

// ValidateEmail validates email format and length
// Uses Go's built-in net/mail parser which follows RFC 5322
func ValidateEmail(email string) error {
	// Check length (RFC 5321: local part max 64, domain max 255, total max 254 with @)
	if len(email) > 254 {
		return errors.New("email address is too long (max 254 characters)")
	}

	if email == "" {
		return errors.New("email address is required")
	}

	// Parse using Go's RFC 5322 compliant parser
	_, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("invalid email address format")
	}

	return nil
}
