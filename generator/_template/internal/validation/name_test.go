package validation

import "testing"

func TestValidateNameRejectsControlCharacters(t *testing.T) {
	for _, name := range []string{"Jane\nDoe", "Jane\rDoe", "Jane\tDoe", "Jane Doe\n"} {
		if err := ValidateName(name); err == nil {
			t.Fatalf("ValidateName(%q) error = nil", name)
		}
	}
}

func TestValidateWorkspaceName(t *testing.T) {
	for _, name := range []string{"A", "Workspace\nB", "Workspace\rB", "Workspace\tB", "\nWorkspace B"} {
		if err := ValidateWorkspaceName(name); err == nil {
			t.Fatalf("ValidateWorkspaceName(%q) error = nil", name)
		}
	}
	if err := ValidateWorkspaceName("Clear Workspace"); err != nil {
		t.Fatalf("ValidateWorkspaceName() error = %v", err)
	}
}
