package generator

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

type wordRow struct {
	Axis  string
	Words []string
}

var wordTable = []wordRow{
	{Axis: "framework", Words: []string{"htmx", "datastar", "headless", "svelte"}},
	{Axis: "database", Words: []string{"sqlite", "postgres"}},
	{Axis: "payment", Words: []string{"polar", "stripe"}},
	{Axis: "mail", Words: []string{"smtp", "resend"}},
	{Axis: "workspaces", Words: []string{"workspaces", "no_workspaces"}},
	{Axis: "storage", Words: []string{"storage", "no_storage"}},
	{Axis: "google", Words: []string{"google", "no_google"}},
	{Axis: "github", Words: []string{"github", "no_github"}},
	{Axis: "blog", Words: []string{"blog", "no_blog"}},
	{Axis: "docs", Words: []string{"docs", "no_docs"}},
	{Axis: "api", Words: []string{"api", "no_api"}},
	{Axis: "mcp", Words: []string{"mcp", "no_mcp"}},
}

var words = func() map[string]string {
	result := make(map[string]string)
	for _, row := range wordTable {
		for _, word := range row.Words {
			if previous, exists := result[word]; exists {
				panic(fmt.Sprintf("condition word %q belongs to %s and %s", word, previous, row.Axis))
			}
			result[word] = row.Axis
		}
	}
	return result
}()

func facts(config Config) map[string]string {
	booleanWord := func(enabled bool, positive, negative string) string {
		if enabled {
			return positive
		}
		return negative
	}
	return map[string]string{
		"framework":  string(config.Framework),
		"database":   string(config.Database),
		"payment":    string(config.Payment),
		"mail":       string(config.Mail),
		"workspaces": booleanWord(config.Workspaces, "workspaces", "no_workspaces"),
		"storage":    booleanWord(config.Storage, "storage", "no_storage"),
		"google":     booleanWord(config.HasGoogleOAuth(), "google", "no_google"),
		"github":     booleanWord(config.HasGitHubOAuth(), "github", "no_github"),
		"blog":       booleanWord(config.HasBlog(), "blog", "no_blog"),
		"docs":       booleanWord(config.HasDocs(), "docs", "no_docs"),
		"api":        booleanWord(config.API, "api", "no_api"),
		"mcp":        booleanWord(config.MCP, "mcp", "no_mcp"),
	}
}

// vendoredDirectories hold vendored sources whose paths are fixed by their
// sync script, so they cannot carry condition words. Their files get these
// words as if the directory carried them.
var vendoredDirectories = map[string][]string{
	"internal/ui/components/": {"htmx", "datastar"},
}

// literalFileNameDirectories hold sources whose file names belong to another
// toolchain, such as "button.svelte", "svelte.config.js" or "app.d.ts". Their
// file names are never parsed for words, so only directory elements carry
// conditions. A final ".tmpl" still renders.
var literalFileNameDirectories = []string{"web.svelte/"}

func hasLiteralFileName(path string) bool {
	for _, directory := range literalFileNameDirectories {
		if strings.HasPrefix(path, directory) {
			return true
		}
	}
	return false
}

// splitConditions removes the condition words from a template path and
// returns the destination and the words each axis allows. Several words of
// one axis in one path element are alternatives: "assets.htmx.datastar"
// selects htmx or Datastar. When an axis appears in several elements, a
// configuration must match every element.
func splitConditions(path string) (string, map[string][]string, error) {
	conditions := make(map[string][]string)
	for directory, vendoredWords := range vendoredDirectories {
		if strings.HasPrefix(filepath.ToSlash(path), directory) {
			for _, word := range vendoredWords {
				conditions[words[word]] = append(conditions[words[word]], word)
			}
		}
	}
	elements := strings.Split(filepath.ToSlash(path), "/")
	literalFileName := hasLiteralFileName(filepath.ToSlash(path))
	for elementIndex, element := range elements {
		if literalFileName && elementIndex == len(elements)-1 {
			continue
		}
		segments := strings.Split(element, ".")
		baseSegments := 1
		if len(segments) > 1 && segments[0] == "" {
			baseSegments = 2
		}
		extensionSegments := 0
		if elementIndex == len(elements)-1 && len(segments) > baseSegments {
			if _, conditionWithoutExtension := words[segments[len(segments)-1]]; !conditionWithoutExtension {
				extensionSegments = 1
				if segments[len(segments)-1] == "tmpl" && len(segments) > baseSegments+1 {
					if _, conditionBeforeTemplate := words[segments[len(segments)-2]]; !conditionBeforeTemplate {
						extensionSegments = 2
					}
				}
			}
		}
		conditionEnd := len(segments) - extensionSegments
		kept := make([]string, 0, len(segments))
		elementConditions := make(map[string][]string)
		for segmentIndex, segment := range segments {
			isCandidate := segmentIndex >= baseSegments && segmentIndex < conditionEnd
			if !isCandidate {
				kept = append(kept, segment)
				continue
			}
			axis, ok := words[segment]
			if !ok {
				if segment == "min" || containsDigit(segment) {
					kept = append(kept, segment)
					continue
				}
				return "", nil, fmt.Errorf("unknown condition word %q", segment)
			}
			if slices.Contains(elementConditions[axis], segment) {
				return "", nil, fmt.Errorf("condition word %q appears twice in %q", segment, element)
			}
			elementConditions[axis] = append(elementConditions[axis], segment)
		}
		for axis, allowed := range elementConditions {
			previous, exists := conditions[axis]
			if !exists {
				conditions[axis] = allowed
				continue
			}
			var both []string
			for _, word := range previous {
				if slices.Contains(allowed, word) {
					both = append(both, word)
				}
			}
			if len(both) == 0 {
				return "", nil, fmt.Errorf("condition axis %q has no word that matches %v and %v", axis, previous, allowed)
			}
			conditions[axis] = both
		}
		elements[elementIndex] = strings.Join(kept, ".")
	}
	return strings.Join(elements, "/"), conditions, nil
}

// matchesConditions reports whether configuration facts select a path with
// these conditions.
func matchesConditions(configFacts map[string]string, conditions map[string][]string) bool {
	for axis, allowed := range conditions {
		if !slices.Contains(allowed, configFacts[axis]) {
			return false
		}
	}
	return true
}

func containsDigit(value string) bool {
	return strings.IndexFunc(value, unicode.IsDigit) >= 0
}

type plannedFile struct {
	Source      string
	Destination string
	Render      bool
}

func (t Template) plan(config Config) ([]plannedFile, error) {
	configFacts := facts(config)
	destinations := make(map[string]string)
	var files []plannedFile
	err := fs.WalkDir(t.files, "_template", func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("_template", source)
		if err != nil {
			return err
		}
		destination, conditions, err := splitConditions(filepath.ToSlash(relative))
		if err != nil {
			return fmt.Errorf("select %s: %w", source, err)
		}
		if !matchesConditions(configFacts, conditions) {
			return nil
		}
		render := strings.HasSuffix(destination, ".tmpl")
		if render {
			destination = strings.TrimSuffix(destination, ".tmpl")
		}
		if previous, exists := destinations[destination]; exists {
			return fmt.Errorf("template sources %s and %s both produce %s", previous, source, destination)
		}
		destinations[destination] = source
		files = append(files, plannedFile{Source: source, Destination: destination, Render: render})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// hasAxis reports whether any template source carries a word of this axis.
// Older template versions predate some axes and cannot render their choices.
func (t Template) hasAxis(axis string) (bool, error) {
	return t.hasCondition(func(conditions map[string][]string) bool {
		_, found := conditions[axis]
		return found
	})
}

// hasWord reports whether any template source carries this word. Older
// template versions predate some values of an existing axis, such as a new
// framework.
func (t Template) hasWord(word string) (bool, error) {
	return t.hasCondition(func(conditions map[string][]string) bool {
		return slices.Contains(conditions[words[word]], word)
	})
}

func (t Template) hasCondition(matches func(map[string][]string) bool) (bool, error) {
	found := false
	err := fs.WalkDir(t.files, "_template", func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel("_template", source)
		if err != nil {
			return err
		}
		_, conditions, err := splitConditions(filepath.ToSlash(relative))
		if err != nil {
			return nil
		}
		found = matches(conditions)
		if found {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("scan template conditions: %w", err)
	}
	return found, nil
}

// requireAxes rejects selections that this template version predates. The
// template tree decides: a version ships a module when a source carries a word
// of its axis, and a framework when a source carries its word.
func (t Template) requireAxes(config Config) error {
	checks := []struct {
		enabled bool
		axis    string
		word    string
		message string
	}{
		// SvelteKit comes first: a tree without it names SvelteKit even when it
		// also predates the JSON API.
		{enabled: config.IsSvelte(), word: string(FrameworkSvelte), message: fmt.Sprintf("SvelteKit needs goilerplate %s or newer", SvelteFirstVersion)},
		{enabled: config.API, axis: "api", message: fmt.Sprintf("The JSON API and Headless need goilerplate %s or newer", APIFirstVersion)},
		{enabled: config.MCP, axis: "mcp", message: fmt.Sprintf("The MCP server needs goilerplate %s or newer", MCPFirstVersion)},
	}
	for _, check := range checks {
		if !check.enabled {
			continue
		}
		var supported bool
		var err error
		if check.word != "" {
			supported, err = t.hasWord(check.word)
		} else {
			supported, err = t.hasAxis(check.axis)
		}
		if err != nil {
			return err
		}
		if !supported {
			return UnsupportedForVersionError{Message: check.message}
		}
	}
	return nil
}
