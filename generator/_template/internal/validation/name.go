package validation

import (
	"errors"
	"strings"
	"unicode"
)

// ValidateName validates profile name
func ValidateName(name string) error {
	if strings.ContainsFunc(name, unicode.IsControl) {
		return errors.New("name cannot contain control characters")
	}
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return errors.New("name is required")
	}

	if len(trimmed) > 100 {
		return errors.New("name is too long (max 100 characters)")
	}
	return nil
}

// ValidateWorkspaceName validates a workspace name.
func ValidateWorkspaceName(name string) error {
	if strings.ContainsFunc(name, unicode.IsControl) {
		return errors.New("workspace name cannot contain control characters")
	}
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < 2 || len(trimmed) > 80 {
		return errors.New("workspace name must contain between 2 and 80 characters")
	}
	return nil
}
