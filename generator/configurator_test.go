package generator

import (
	"encoding/base64"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestConfiguratorManifestContainsEveryValidSelection(t *testing.T) {
	manifest, err := BuildConfiguratorManifest("3.0.0-test")
	if err != nil {
		t.Fatalf("BuildConfiguratorManifest() error = %v", err)
	}
	selections := configuratorSelections()
	headlessSelections := 0
	mcpSelections := 0
	for _, config := range selections {
		if config.MCP {
			mcpSelections++
			continue
		}
		if slices.Contains(configuratorFrontends(config), FrameworkHeadless) {
			headlessSelections++
		}
	}
	if len(selections) != 1536 || mcpSelections != 512 || headlessSelections != 128 || len(manifest.Trees) != len(selections)-mcpSelections+headlessSelections {
		t.Fatalf("selections = %d, mcp = %d, headless = %d, trees = %d, want 1536, 512, 128 and 1152", len(selections), mcpSelections, headlessSelections, len(manifest.Trees))
	}
	if len(manifest.Frontends) != 4 {
		t.Fatalf("frontends = %d, want 4", len(manifest.Frontends))
	}
	_, ok := manifest.FrontendRemovals["svelte"]
	if !ok || len(manifest.FrontendRemovals) != 1 {
		t.Fatalf("frontend removals = %v, want svelte", manifest.FrontendRemovals)
	}
	if _, ok := manifest.Modules["mcp"]; !ok || len(manifest.Modules) != 1 {
		t.Fatalf("modules = %v, want mcp", manifest.Modules)
	}
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range selections {
		key, err := ConfiguratorSelectionKey(config)
		if err != nil {
			t.Fatal(err)
		}
		for _, framework := range configuratorFrontends(config) {
			config.Framework = framework
			treeKey := key
			if framework == FrameworkHeadless || framework == FrameworkSvelte {
				treeKey = ConfiguratorHeadlessKey(key)
			}
			actual := decodeConfiguratorTree(t, manifest, treeKey, framework, config.MCP)
			if framework == FrameworkSvelte {
				actual = append(actual, decodeConfiguratorWebTree(t, manifest, config)...)
				slices.Sort(actual)
			}
			want, err := template.PreviewFiles(config)
			if err != nil {
				t.Fatalf("PreviewFiles(%q, %s) error = %v", key, framework, err)
			}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("manifest tree %q with %s differs from generator preview", key, framework)
			}
		}
	}
}

func TestConfiguratorSelectionKeyNormalizesMultiSelectOrder(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.OAuth = []OAuth{OAuthGitHub, OAuthGoogle, OAuthGitHub}
	config.Content = []Content{ContentDocs, ContentBlog, ContentDocs}
	key, err := ConfiguratorSelectionKey(config)
	if err != nil {
		t.Fatal(err)
	}
	if key != "sqlite|stripe|smtp|true|google,github|true|blog,docs" {
		t.Fatalf("key = %q", key)
	}
}

func TestConfiguratorSelectionKeyMarksAPI(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.API = true
	key, err := ConfiguratorSelectionKey(config)
	if err != nil {
		t.Fatal(err)
	}
	if key != "sqlite|stripe|smtp|true|google,github|true|blog,docs|api" {
		t.Fatalf("key = %q", key)
	}
}

func TestConfiguratorSelectionKeyLeavesMCPToTheModuleDelta(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.API = true
	withoutMCP, err := ConfiguratorSelectionKey(config)
	if err != nil {
		t.Fatal(err)
	}
	config.MCP = true
	withMCP, err := ConfiguratorSelectionKey(config)
	if err != nil {
		t.Fatal(err)
	}
	if withMCP != withoutMCP {
		t.Fatalf("key with MCP = %q, without = %q", withMCP, withoutMCP)
	}
	config.API = false
	_, err = ConfiguratorSelectionKey(config)
	if err == nil || !strings.Contains(err.Error(), "MCP requires the JSON API") {
		t.Fatalf("key for MCP without the API error = %v", err)
	}
}

func TestConfiguratorFrontendsOfferHeadlessAndSvelteKitOnlyWithAPIAndNoContent(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.API = true
	got := len(configuratorFrontends(config))
	if got != 2 {
		t.Fatalf("frontends with content = %d, want 2", got)
	}
	config.Content = nil
	frontends := configuratorFrontends(config)
	if !slices.Equal(frontends, []Framework{FrameworkHTMX, FrameworkDatastar, FrameworkHeadless, FrameworkSvelte}) {
		t.Fatalf("frontends without content = %v", frontends)
	}
	config.API = false
	got = len(configuratorFrontends(config))
	if got != 2 {
		t.Fatalf("frontends without api = %d, want 2", got)
	}
}

func decodeConfiguratorTree(t *testing.T, manifest ConfiguratorManifest, key string, framework Framework, mcp bool) []string {
	t.Helper()
	encoded, ok := manifest.Trees[key]
	if !ok {
		t.Fatalf("missing tree %q", key)
	}
	bits, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode tree %q: %v", key, err)
	}
	frontendBits, err := base64.RawStdEncoding.DecodeString(manifest.Frontends[string(framework)])
	if err != nil {
		t.Fatalf("decode %s frontend: %v", framework, err)
	}
	for index := range bits {
		bits[index] |= frontendBits[index]
	}
	removals, ok := manifest.FrontendRemovals[string(framework)]
	if ok {
		removalBits, err := base64.RawStdEncoding.DecodeString(removals)
		if err != nil {
			t.Fatalf("decode %s frontend removals: %v", framework, err)
		}
		for index := range bits {
			bits[index] &^= removalBits[index]
		}
	}
	if mcp {
		moduleBits, err := base64.RawStdEncoding.DecodeString(manifest.Modules["mcp"])
		if err != nil {
			t.Fatalf("decode mcp module: %v", err)
		}
		for index := range bits {
			bits[index] |= moduleBits[index]
		}
	}
	var paths []string
	for index, path := range manifest.Paths {
		if bits[index/8]&(1<<(index%8)) != 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

func decodeConfiguratorWebTree(t *testing.T, manifest ConfiguratorManifest, config Config) []string {
	t.Helper()
	key := configuratorWebKey(config, manifest.Web.Axes)
	encoded, ok := manifest.Web.Trees[key]
	if !ok {
		t.Fatalf("missing web tree %q", key)
	}
	bits, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode web tree %q: %v", key, err)
	}
	var paths []string
	for index, path := range manifest.Web.Paths {
		if bits[index/8]&(1<<(index%8)) != 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

func TestConfiguratorWebKeyUsesTheWebAxesInOrder(t *testing.T) {
	config := fullSQLiteHTMXConfig()
	config.Framework = FrameworkSvelte
	config.API = true
	config.Content = nil
	key := configuratorWebKey(config, []string{"workspaces", "storage"})
	if key != "workspaces|storage" {
		t.Fatalf("key = %q", key)
	}
	key = configuratorWebKey(config, nil)
	if key != "" {
		t.Fatalf("key without axes = %q", key)
	}
	template, err := CurrentTemplate("3.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	axes, err := template.webAxes()
	if err != nil {
		t.Fatal(err)
	}
	for _, axis := range axes {
		if axis == "framework" {
			t.Fatalf("web axes include the framework: %v", axes)
		}
	}
}
