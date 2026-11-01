package generator

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type ConfiguratorManifest struct {
	Paths     []string          `json:"paths"`
	Trees     map[string]string `json:"trees"`
	Frontends map[string]string `json:"frontends"`
	// FrontendRemovals holds the paths a frontend drops from the tree it
	// builds on. A SvelteKit tree is the Headless tree of the same key plus
	// Frontends["svelte"] minus FrontendRemovals["svelte"], plus its Web tree.
	FrontendRemovals map[string]string `json:"frontend_removals"`
	// Web lists the SvelteKit app files under web/, keyed by the facts of the
	// axes the web.svelte sources use.
	Web ConfiguratorWeb `json:"web"`
	// Modules holds the paths an add-on module adds to any tree it can join.
	// The MCP server adds the same paths to every API selection and frontend,
	// so its trees are not stored twice.
	Modules map[string]string `json:"modules"`
}

// ConfiguratorWeb holds the SvelteKit app trees. Axes names the axes whose
// facts, joined with "|" in this order, form the key of Trees.
type ConfiguratorWeb struct {
	Axes  []string          `json:"axes"`
	Paths []string          `json:"paths"`
	Trees map[string]string `json:"trees"`
}

// configuratorWebPrefix is the directory of the SvelteKit app in a project.
const configuratorWebPrefix = "web/"

func BuildConfiguratorManifest(templateVersion string) (ConfiguratorManifest, error) {
	template, err := CurrentTemplate(templateVersion)
	if err != nil {
		return ConfiguratorManifest{}, err
	}

	filesByKey := make(map[string][]string)
	allPaths := make(map[string]struct{})
	frontendFiles := make(map[Framework][]string)
	var mcpFiles []string
	mcpSeen := false
	webAxes, err := template.webAxes()
	if err != nil {
		return ConfiguratorManifest{}, err
	}
	webFilesByKey := make(map[string][]string)
	var svelteAdded, svelteRemoved []string
	svelteSeen := false
	for _, config := range configuratorSelections() {
		key, err := ConfiguratorSelectionKey(config)
		if err != nil {
			return ConfiguratorManifest{}, err
		}
		if config.MCP {
			for _, framework := range configuratorFrontends(config) {
				delta, err := configuratorMCPDelta(template, config, framework)
				if err != nil {
					return ConfiguratorManifest{}, fmt.Errorf("preview %s %s with MCP: %w", key, framework, err)
				}
				if mcpSeen && !slices.Equal(mcpFiles, delta) {
					return ConfiguratorManifest{}, errors.New("mcp paths vary by selection")
				}
				mcpFiles = delta
				mcpSeen = true
			}
			for _, path := range mcpFiles {
				allPaths[path] = struct{}{}
			}
			continue
		}
		files, err := template.PreviewFiles(config)
		if err != nil {
			return ConfiguratorManifest{}, fmt.Errorf("preview %s: %w", key, err)
		}
		datastarConfig := config
		datastarConfig.Framework = FrameworkDatastar
		datastarFiles, err := template.PreviewFiles(datastarConfig)
		if err != nil {
			return ConfiguratorManifest{}, fmt.Errorf("preview %s Datastar: %w", key, err)
		}
		common, htmxOnly, datastarOnly := splitConfiguratorFiles(files, datastarFiles)
		files = common
		for framework, delta := range map[Framework][]string{
			FrameworkHTMX:     htmxOnly,
			FrameworkDatastar: datastarOnly,
		} {
			if previous, ok := frontendFiles[framework]; ok && !slices.Equal(previous, delta) {
				return ConfiguratorManifest{}, fmt.Errorf("%s frontend paths vary by selection", framework)
			}
			frontendFiles[framework] = delta
		}
		filesByKey[key] = files
		for _, path := range files {
			allPaths[path] = struct{}{}
		}
		if !slices.Contains(configuratorFrontends(config), FrameworkHeadless) {
			continue
		}
		headlessConfig := config
		headlessConfig.Framework = FrameworkHeadless
		headlessFiles, err := template.PreviewFiles(headlessConfig)
		if err != nil {
			return ConfiguratorManifest{}, fmt.Errorf("preview %s Headless: %w", key, err)
		}
		headlessKey := ConfiguratorHeadlessKey(key)
		filesByKey[headlessKey] = headlessFiles
		for _, path := range headlessFiles {
			allPaths[path] = struct{}{}
		}
		if !slices.Contains(configuratorFrontends(config), FrameworkSvelte) {
			continue
		}
		svelteConfig := config
		svelteConfig.Framework = FrameworkSvelte
		svelteFiles, err := template.PreviewFiles(svelteConfig)
		if err != nil {
			return ConfiguratorManifest{}, fmt.Errorf("preview %s SvelteKit: %w", key, err)
		}
		var webFiles, backendFiles []string
		for _, path := range svelteFiles {
			if strings.HasPrefix(path, configuratorWebPrefix) {
				webFiles = append(webFiles, path)
			} else {
				backendFiles = append(backendFiles, path)
			}
		}
		_, added, removed := splitConfiguratorFiles(backendFiles, headlessFiles)
		if svelteSeen && (!slices.Equal(svelteAdded, added) || !slices.Equal(svelteRemoved, removed)) {
			return ConfiguratorManifest{}, errors.New("svelte frontend paths vary by selection")
		}
		svelteAdded, svelteRemoved, svelteSeen = added, removed, true
		webKey := configuratorWebKey(svelteConfig, webAxes)
		previous, ok := webFilesByKey[webKey]
		if ok && !slices.Equal(previous, webFiles) {
			return ConfiguratorManifest{}, fmt.Errorf("svelte web paths vary within web key %q", webKey)
		}
		webFilesByKey[webKey] = webFiles
	}
	frontendFiles[FrameworkHeadless] = nil
	frontendFiles[FrameworkSvelte] = svelteAdded
	for _, files := range frontendFiles {
		for _, path := range files {
			allPaths[path] = struct{}{}
		}
	}

	paths := make([]string, 0, len(allPaths))
	for path := range allPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	pathIndexes := make(map[string]int, len(paths))
	for index, path := range paths {
		pathIndexes[path] = index
	}

	encode := func(files []string) string {
		bits := make([]byte, (len(paths)+7)/8)
		for _, path := range files {
			index := pathIndexes[path]
			bits[index/8] |= 1 << (index % 8)
		}
		return base64.RawStdEncoding.EncodeToString(bits)
	}

	trees := make(map[string]string, len(filesByKey))
	for key, files := range filesByKey {
		trees[key] = encode(files)
	}
	frontends := make(map[string]string, len(frontendFiles))
	for framework, files := range frontendFiles {
		frontends[string(framework)] = encode(files)
	}

	webPathSet := make(map[string]struct{})
	for _, files := range webFilesByKey {
		for _, path := range files {
			webPathSet[path] = struct{}{}
		}
	}
	webPaths := make([]string, 0, len(webPathSet))
	for path := range webPathSet {
		webPaths = append(webPaths, path)
	}
	sort.Strings(webPaths)
	webPathIndexes := make(map[string]int, len(webPaths))
	for index, path := range webPaths {
		webPathIndexes[path] = index
	}
	webTrees := make(map[string]string, len(webFilesByKey))
	for key, files := range webFilesByKey {
		bits := make([]byte, (len(webPaths)+7)/8)
		for _, path := range files {
			index := webPathIndexes[path]
			bits[index/8] |= 1 << (index % 8)
		}
		webTrees[key] = base64.RawStdEncoding.EncodeToString(bits)
	}

	return ConfiguratorManifest{
		Paths:            paths,
		Trees:            trees,
		Frontends:        frontends,
		FrontendRemovals: map[string]string{string(FrameworkSvelte): encode(svelteRemoved)},
		Modules:          map[string]string{"mcp": encode(mcpFiles)},
		Web: ConfiguratorWeb{
			Axes:  webAxes,
			Paths: webPaths,
			Trees: webTrees,
		},
	}, nil
}

// webAxes lists, in word table order, the axes other than the framework that
// the web.svelte sources select on.
func (t Template) webAxes() ([]string, error) {
	used := make(map[string]bool)
	err := fs.WalkDir(t.files, "_template/web.svelte", func(source string, entry fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, fs.ErrNotExist) {
			return fs.SkipAll
		}
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
		_, conditions, err := splitConditions(filepath.ToSlash(relative))
		if err != nil {
			return fmt.Errorf("select %s: %w", source, err)
		}
		for axis := range conditions {
			used[axis] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan web conditions: %w", err)
	}
	axes := []string{}
	for _, row := range wordTable {
		if row.Axis != "framework" && used[row.Axis] {
			axes = append(axes, row.Axis)
		}
	}
	return axes, nil
}

// configuratorWebKey names the SvelteKit app tree of a config: the facts of
// the web axes joined with "|".
func configuratorWebKey(config Config, axes []string) string {
	configFacts := facts(config)
	values := make([]string, len(axes))
	for index, axis := range axes {
		values[index] = configFacts[axis]
	}
	return strings.Join(values, "|")
}

// configuratorMCPDelta returns the paths MCP adds to a selection with one
// frontend. MCP may only add paths.
func configuratorMCPDelta(template Template, config Config, framework Framework) ([]string, error) {
	config.Framework = framework
	withMCP, err := template.PreviewFiles(config)
	if err != nil {
		return nil, err
	}
	config.MCP = false
	withoutMCP, err := template.PreviewFiles(config)
	if err != nil {
		return nil, err
	}
	_, added, removed := splitConfiguratorFiles(withMCP, withoutMCP)
	if len(removed) != 0 {
		return nil, fmt.Errorf("MCP removes %v", removed)
	}
	return added, nil
}

// ConfiguratorSelectionKey names the module selection of a config. The
// frontend is not part of the key: htmx and Datastar trees are the keyed tree
// plus their frontend delta, and a Headless tree lives under
// ConfiguratorHeadlessKey because Headless drops paths that htmx and Datastar
// share. MCP is not part of the key either: an MCP tree is the tree without
// MCP plus Modules["mcp"].
func ConfiguratorSelectionKey(config Config) (string, error) {
	normalized, err := config.normalized()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	key := strings.Join([]string{
		string(normalized.Database),
		string(normalized.Payment),
		string(normalized.Mail),
		fmt.Sprintf("%t", normalized.Workspaces),
		joinOAuth(normalized.OAuth),
		fmt.Sprintf("%t", normalized.Storage),
		joinContent(normalized.Content),
	}, "|")
	if normalized.API {
		key += "|api"
	}
	return key, nil
}

// ConfiguratorHeadlessKey returns the manifest key of the complete Headless
// tree for a selection key. Its frontend delta is empty. SvelteKit builds on
// the same tree.
func ConfiguratorHeadlessKey(selectionKey string) string {
	return selectionKey + "|headless"
}

// configuratorFrontends lists the frontends a selection can render with.
// Headless and SvelteKit require the JSON API and never ship blog or docs.
func configuratorFrontends(config Config) []Framework {
	frameworks := []Framework{FrameworkHTMX, FrameworkDatastar}
	if config.API && len(config.Content) == 0 {
		frameworks = append(frameworks, FrameworkHeadless, FrameworkSvelte)
	}
	return frameworks
}

func configuratorSelections() []Config {
	var selections []Config
	for _, database := range []Database{DatabaseSQLite, DatabasePostgres} {
		for _, payment := range []Payment{PaymentStripe, PaymentPolar} {
			for _, mail := range []Mail{MailSMTP, MailResend} {
				for _, workspaces := range []bool{false, true} {
					for _, oauth := range [][]OAuth{nil, {OAuthGoogle}, {OAuthGitHub}, {OAuthGoogle, OAuthGitHub}} {
						for _, storage := range []bool{false, true} {
							for _, content := range [][]Content{nil, {ContentBlog}, {ContentDocs}, {ContentBlog, ContentDocs}} {
								selections = append(selections, Config{
									ProjectName: "My App",
									ModulePath:  "example.com/my-app",
									Framework:   FrameworkHTMX,
									Database:    database,
									Payment:     payment,
									Mail:        mail,
									Workspaces:  workspaces,
									OAuth:       oauth,
									Storage:     storage,
									Content:     content,
								})
							}
						}
					}
				}
			}
		}
	}
	withAPI := make([]Config, 0, 3*len(selections))
	for _, selection := range selections {
		apiSelection := selection
		apiSelection.API = true
		mcpSelection := apiSelection
		mcpSelection.MCP = true
		withAPI = append(withAPI, selection, apiSelection, mcpSelection)
	}
	return withAPI
}

func splitConfiguratorFiles(left, right []string) (common, leftOnly, rightOnly []string) {
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, path := range left {
		leftSet[path] = struct{}{}
	}
	for _, path := range right {
		rightSet[path] = struct{}{}
	}
	for _, path := range left {
		if _, ok := rightSet[path]; ok {
			common = append(common, path)
		} else {
			leftOnly = append(leftOnly, path)
		}
	}
	for _, path := range right {
		if _, ok := leftSet[path]; !ok {
			rightOnly = append(rightOnly, path)
		}
	}
	return common, leftOnly, rightOnly
}

func joinOAuth(values []OAuth) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = string(value)
	}
	return strings.Join(parts, ",")
}

func joinContent(values []Content) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = string(value)
	}
	return strings.Join(parts, ",")
}
