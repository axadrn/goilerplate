package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const LockSchemaVersion = 1

var ErrInvalidConfig = errors.New("invalid generator config")

// ErrUnsupportedForVersion reports a valid selection that the requested
// template version does not ship.
var ErrUnsupportedForVersion = errors.New("selection not supported by this template version")

// APIFirstVersion is the first release whose template ships the JSON API and
// Headless.
const APIFirstVersion = "v3.1.0"

// MCPFirstVersion is the first release whose template ships the MCP server.
const MCPFirstVersion = "v3.1.0"

// SvelteFirstVersion is the first release whose template ships the SvelteKit
// frontend.
const SvelteFirstVersion = "v3.1.0"

// UnsupportedForVersionError names a selection the requested template version
// does not ship. It matches ErrUnsupportedForVersion with errors.Is.
type UnsupportedForVersionError struct {
	// Message is a sentence for the person who asked, such as "The MCP server
	// needs goilerplate v3.1.0 or newer".
	Message string
}

func (e UnsupportedForVersionError) Error() string {
	return ErrUnsupportedForVersion.Error() + ": " + e.Message
}

func (e UnsupportedForVersionError) Unwrap() error { return ErrUnsupportedForVersion }

type LockFile struct {
	SchemaVersion   int    `json:"schema_version"`
	TemplateVersion string `json:"template_version"`
	Config          Config `json:"config"`
}

func Generate(destination, templateVersion string, config Config) error {
	template, err := CurrentTemplate(templateVersion)
	if err != nil {
		return err
	}
	return template.Generate(destination, config)
}

// Generate renders this template version into an empty destination directory.
func (t Template) Generate(destination string, config Config) error {
	normalized, err := config.normalized()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	err = t.requireAxes(normalized)
	if err != nil {
		return err
	}
	if err := requireEmptyDirectory(destination); err != nil {
		return err
	}
	files, err := t.plan(normalized)
	if err != nil {
		return fmt.Errorf("plan generated files: %w", err)
	}
	for _, file := range files {
		if file.Render {
			if err := t.renderPlannedFile(destination, file, normalized); err != nil {
				return fmt.Errorf("render %s: %w", file.Source, err)
			}
			continue
		}
		if err := t.copyPlannedFile(destination, file, normalized); err != nil {
			return fmt.Errorf("copy %s: %w", file.Source, err)
		}
	}
	lock := LockFile{
		SchemaVersion:   LockSchemaVersion,
		TemplateVersion: t.version,
		Config:          normalized,
	}
	if err := writeJSON(filepath.Join(destination, "goilerplate.lock"), lock); err != nil {
		return fmt.Errorf("write lock file: %w", err)
	}
	return nil
}

func requireEmptyDirectory(destination string) error {
	entries, err := os.ReadDir(destination)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(destination, 0o755)
	}
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("destination must be empty")
	}
	return nil
}

func (t Template) copyPlannedFile(destination string, file plannedFile, config Config) error {
	timestampType := "TIMESTAMP"
	if config.Database == DatabasePostgres {
		timestampType = "TIMESTAMPTZ"
	}
	target := filepath.Join(destination, filepath.FromSlash(file.Destination))
	content, err := t.readFile(file.Source)
	if err != nil {
		return err
	}
	content = bytes.ReplaceAll(content, []byte("{{.ModulePath}}"), []byte(config.ModulePath))
	content = bytes.ReplaceAll(content, []byte("{{.ProjectNameGo}}"), []byte(strconv.Quote(config.ProjectName)))
	content = bytes.ReplaceAll(content, []byte("{{.DatabaseTimestamp}}"), []byte(timestampType))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, content, 0o644)
}

func (t Template) renderPlannedFile(destination string, file plannedFile, config Config) error {
	target := filepath.Join(destination, filepath.FromSlash(file.Destination))
	content, err := t.readFile(file.Source)
	if err != nil {
		return err
	}
	parsed, err := template.New(file.Source).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, config); err != nil {
		return err
	}
	output := rendered.Bytes()
	if strings.HasSuffix(target, ".go") {
		output, err = format.Source(output)
		if err != nil {
			return fmt.Errorf("format rendered Go file %q: %w", file.Source, err)
		}
	} else if filepath.Base(target) == "go.mod" {
		output = formatModuleFile(output)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, output, 0o644)
}

func formatModuleFile(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	formatted := make([]string, 0, len(lines))
	inRequireBlock := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "require (" {
			inRequireBlock = true
			formatted = append(formatted, line)
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			formatted = append(formatted, line)
			continue
		}
		if inRequireBlock && strings.TrimSpace(line) == "" {
			continue
		}
		formatted = append(formatted, line)
	}
	return []byte(strings.TrimRight(strings.Join(formatted, "\n"), "\n") + "\n")
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
}
