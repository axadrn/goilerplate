package generator

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	templCSSPath   = "_template/assets.htmx.datastar/css/globals.htmx.css"
	svelteCSSPath  = "_template/web.svelte/src/app.css"
	svelteRoot     = "_template/web.svelte"
	sveltePackage  = "_template/web.svelte/package.json"
	svelteLockfile = "_template/web.svelte/pnpm-lock.yaml"
)

func TestSvelteThemeMatchesTheTemplApp(t *testing.T) {
	templ := readTemplateFile(t, templCSSPath)
	svelte := readTemplateFile(t, svelteCSSPath)
	for _, block := range []string{"@theme inline {", ":root {", ".dark {", "@layer base {"} {
		want := cssBlock(t, templCSSPath, templ, block)
		got := cssBlock(t, svelteCSSPath, svelte, block)
		if got != want {
			t.Errorf("%s %s differs from the templ app:\n%s\nwant:\n%s", svelteCSSPath, block, got, want)
		}
	}
	for _, line := range []string{"@custom-variant dark (&:where(.dark, .dark *));", `@import "tailwindcss";`, `@import "tw-animate-css";`, `@import "shadcn-svelte/tailwind.css";`} {
		if !strings.Contains(svelte, line) {
			t.Errorf("%s misses %s", svelteCSSPath, line)
		}
	}
	// SvelteKit pages show pending requests with aria-busy, like Datastar.
	if !strings.Contains(svelte, `:is(button[aria-busy], form[aria-busy] button[type="submit"])`) {
		t.Errorf("%s misses the aria-busy pending styles", svelteCSSPath)
	}
}

func TestSvelteFontsMatchTheTemplApp(t *testing.T) {
	templFonts := fontFaces(t, readTemplateFile(t, templCSSPath), "/assets/fonts/")
	svelteFonts := fontFaces(t, readTemplateFile(t, svelteCSSPath), "/fonts/")
	if len(svelteFonts) == 0 || !slices.Equal(templFonts, svelteFonts) {
		t.Fatalf("SvelteKit font faces = %v, want %v", svelteFonts, templFonts)
	}
	for _, face := range templFonts {
		file := face[strings.LastIndex(face, " ")+1:]
		want := readTemplateFile(t, "_template/assets.htmx.datastar/fonts/"+file)
		got := readTemplateFile(t, svelteRoot+"/static/fonts/"+file)
		if got != want {
			t.Errorf("web.svelte/static/fonts/%s differs from the templ app font", file)
		}
	}
	for _, license := range []string{"bricolage/OFL.txt", "geist/README.md"} {
		want := readTemplateFile(t, "_template/assets.htmx.datastar/fonts/"+license)
		got := readTemplateFile(t, svelteRoot+"/static/fonts/"+license)
		if got != want {
			t.Errorf("web.svelte/static/fonts/%s differs from the templ app", license)
		}
	}
}

func TestSvelteTemplateHasNoBuildOutput(t *testing.T) {
	var found []string
	err := filepath.WalkDir(svelteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() && (name == "node_modules" || name == ".svelte-kit") {
			found = append(found, path)
			return fs.SkipDir
		}
		if !entry.IsDir() && strings.HasPrefix(path, svelteRoot+"/build/") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(found, []string{svelteRoot + "/build/.gitkeep"}) {
		t.Errorf("web.svelte holds build output or dependencies: %v, want only build/.gitkeep", found)
	}
	// pnpm build empties build/ and copies static/ into it, so static/.gitkeep
	// keeps build/.gitkeep in place after every build.
	_, err = os.Stat(svelteRoot + "/static/.gitkeep")
	if err != nil {
		t.Errorf("web.svelte/static/.gitkeep is missing: %v", err)
	}
}

var (
	rawSvelteControl = regexp.MustCompile(`<(?:button|input|select|textarea|table|dialog)\b`)
	svelteFetch      = regexp.MustCompile(`\bfetch\(`)
)

func TestSvelteSourcesUseShadcnPrimitives(t *testing.T) {
	vendored := []string{svelteRoot + "/src/lib/components/ui/", svelteRoot + "/src/lib/hooks/", svelteRoot + "/src/lib/utils.ts"}
	err := filepath.WalkDir(svelteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasPrefix(path, svelteRoot+"/src/") && path != svelteRoot+"/static/theme.js" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(content)
		if strings.Contains(source, "localStorage") && path != svelteRoot+"/static/theme.js" {
			t.Errorf("%s uses localStorage; only static/theme.js keeps the theme there", path)
		}
		if slices.ContainsFunc(vendored, func(prefix string) bool { return strings.HasPrefix(path, prefix) }) {
			return nil
		}
		if svelteFetch.MatchString(source) && path != svelteRoot+"/src/lib/api.ts" {
			t.Errorf("%s calls fetch; requests go through src/lib/api.ts", path)
		}
		if filepath.Ext(path) != ".svelte" {
			return nil
		}
		match := rawSvelteControl.FindString(source)
		if match != "" {
			t.Errorf("%s contains a raw %s> instead of a shadcn-svelte component", path, match)
		}
		if strings.Contains(source, "{@html") {
			t.Errorf("%s renders raw HTML with {@html}", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSveltePackagePinsExactVersions(t *testing.T) {
	var manifest struct {
		PackageManager  string            `json:"packageManager"`
		Engines         map[string]string `json:"engines"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	err := json.Unmarshal([]byte(readTemplateFile(t, sveltePackage)), &manifest)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.PackageManager != "pnpm@12.9.1" {
		t.Errorf("packageManager = %q, want pnpm@12.9.1", manifest.PackageManager)
	}
	if manifest.Engines["node"] != ">=22.17" {
		t.Errorf("engines.node = %q, want >=22.17", manifest.Engines["node"])
	}
	if len(manifest.Dependencies) != 0 {
		t.Errorf("dependencies = %v, want everything in devDependencies: the build output is static", manifest.Dependencies)
	}
	_, found := manifest.DevDependencies["mode-watcher"]
	if found {
		t.Error("devDependencies has mode-watcher; static/theme.js owns the theme")
	}
	for name, want := range map[string]string{
		"svelte":                   "5.57.1",
		"@sveltejs/kit":            "3.0.0",
		"@sveltejs/adapter-static": "4.0.0",
		"vite":                     "8.3.2",
		"tailwindcss":              "4.3.3",
		"shadcn-svelte":            "1.7.0",
		"bits-ui":                  "2.19.5",
		"typescript":               "6.0.3",
	} {
		got := manifest.DevDependencies[name]
		if got != want {
			t.Errorf("devDependencies[%q] = %q, want %q", name, got, want)
		}
	}
	exact := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	lockfile := readTemplateFile(t, svelteLockfile)
	for name, version := range manifest.DevDependencies {
		if !exact.MatchString(version) {
			t.Errorf("devDependencies[%q] = %q, want an exact version", name, version)
		}
		key := name
		if strings.HasPrefix(name, "@") {
			key = "'" + name + "'"
		}
		if !strings.Contains(lockfile, "\n      "+key+":\n        specifier: "+version+"\n") {
			t.Errorf("pnpm-lock.yaml does not pin %s %s; run generator/scripts/sync-shadcn-svelte.sh", name, version)
		}
	}
}

func readTemplateFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")))
}

// cssBlock returns the block that starts with header, up to its closing brace.
func cssBlock(t *testing.T, path, css, header string) string {
	t.Helper()
	start := strings.Index(css, "\n"+header)
	if start < 0 {
		t.Fatalf("%s has no %s block", path, header)
	}
	depth := 0
	for index := start + 1; index < len(css); index++ {
		switch css[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[start+1 : index+1]
			}
		}
	}
	t.Fatalf("%s %s block is not closed", path, header)
	return ""
}

var fontFaceRule = regexp.MustCompile(`@font-face \{\s*font-family: "([^"]+)";\s*src: url\("([^"]+)"\) format\("([^"]+)"\);\s*font-weight: ([^;]+);\s*font-display: ([^;]+);\s*\}`)

// fontFaces lists each @font-face as "family weight display format file", with
// the file relative to prefix.
func fontFaces(t *testing.T, css, prefix string) []string {
	t.Helper()
	var faces []string
	for _, match := range fontFaceRule.FindAllStringSubmatch(css, -1) {
		file, found := strings.CutPrefix(match[2], prefix)
		if !found {
			t.Fatalf("font %s is not under %s", match[2], prefix)
		}
		faces = append(faces, strings.Join([]string{match[1], match[4], match[5], match[3], file}, " "))
	}
	return faces
}
