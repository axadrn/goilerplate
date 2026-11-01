package validation

import (
	"errors"
	"strings"
)

// ValidatePassword validates password strength
// Enforces NIST recommendations: minimum 12 characters, blocks common patterns
func ValidatePassword(password string) error {
	// Minimum length: 12 characters (NIST recommendation)
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}

	// Maximum length: 72 bytes (bcrypt limitation)
	// bcrypt silently truncates passwords longer than 72 bytes, which is a security risk
	if len(password) > 72 {
		return errors.New("password must not exceed 72 characters")
	}

	// Check for common/weak patterns
	lower := strings.ToLower(password)
	commonPatterns := []string{
		"password", "123456", "qwerty", "admin", "letmein",
		"welcome", "monkey", "dragon", "master", "sunshine",
	}

	for _, pattern := range commonPatterns {
		if strings.Contains(lower, pattern) {
			return errors.New("password is too common, please choose a stronger one")
		}
	}

	return nil
}
