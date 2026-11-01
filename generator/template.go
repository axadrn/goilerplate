package generator

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"golang.org/x/mod/semver"
)

//go:embed all:_template
var embeddedTemplate embed.FS

// Template is one immutable version of the master template.
type Template struct {
	version string
	files   fs.FS
}

// CurrentTemplate returns the master template embedded in this binary.
func CurrentTemplate(templateVersion string) (Template, error) {
	return NewTemplate(templateVersion, embeddedTemplate)
}

// NewTemplate creates a generator from a template filesystem.
func NewTemplate(templateVersion string, files fs.FS) (Template, error) {
	if strings.TrimSpace(templateVersion) == "" {
		return Template{}, errors.New("template version is required")
	}
	if files == nil {
		return Template{}, errors.New("template files are required")
	}
	return Template{
		version: templateVersion,
		files:   files,
	}, nil
}

// ValidateVersion requires the canonical Git tag form used by releases.
func ValidateVersion(version string) error {
	if version == "" {
		return errors.New("version is required")
	}
	if strings.TrimSpace(version) != version || !semver.IsValid(version) || semver.Canonical(version) != version {
		return fmt.Errorf("version %q must look like v3.0.0", version)
	}
	return nil
}

func (t Template) readFile(name string) ([]byte, error) {
	content, err := fs.ReadFile(t.files, name)
	if err != nil {
		return nil, fmt.Errorf("read template file %s: %w", name, err)
	}
	return content, nil
}
