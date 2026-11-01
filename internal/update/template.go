package update

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing/fstest"

	"golang.org/x/mod/semver"
)

// DefaultProxy is the public Go module proxy. Every release of goilerplate
// carries its template there, so update needs no goilerplate server.
const DefaultProxy = "https://proxy.golang.org"

// FirstUpdatableMajor is the first major version whose releases carry the
// template in this public module.
const FirstUpdatableMajor = "v4"

const (
	repositoryModule = "github.com/axadrn/goilerplate"
	templateDir      = "generator/_template/"
	maxModuleZipSize = 128 << 20
	maxTemplateSize  = 256 << 20
)

// ErrVersionNotFound reports a template version the proxy does not have.
var ErrVersionNotFound = errors.New("template version not found")

// ModulePath returns the module path of a goilerplate release, such as
// github.com/axadrn/goilerplate/v4 for v4.1.0.
func ModulePath(version string) string {
	major := semver.Major(version)
	if major == "v0" || major == "v1" {
		return repositoryModule
	}
	return repositoryModule + "/" + major
}

// FetchTemplate downloads the release of version from the module proxy and
// returns its template tree, rooted at _template/ as the generator expects.
func FetchTemplate(ctx context.Context, client *http.Client, proxy, version string) (fs.FS, error) {
	if !semver.IsValid(version) {
		return nil, fmt.Errorf("invalid template version %q", version)
	}
	module := ModulePath(version)
	endpoint := fmt.Sprintf("%s/%s/@v/%s.zip", strings.TrimRight(proxy, "/"), module, version)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download goilerplate %s: %w", version, err)
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone:
		return nil, fmt.Errorf("%w: %s", ErrVersionNotFound, version)
	case response.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("download goilerplate %s: the module proxy returned %s", version, response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxModuleZipSize+1))
	if err != nil {
		return nil, fmt.Errorf("download goilerplate %s: %w", version, err)
	}
	if len(content) > maxModuleZipSize {
		return nil, fmt.Errorf("download goilerplate %s: the module is too large", version)
	}
	return readTemplate(content, module+"@"+version+"/")
}

func readTemplate(content []byte, prefix string) (fs.FS, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("open module zip: %w", err)
	}
	files := make(fstest.MapFS)
	remaining := int64(maxTemplateSize)
	for _, file := range archive.File {
		name, ok := strings.CutPrefix(file.Name, prefix+templateDir)
		if !ok || name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		if !fs.ValidPath(name) {
			return nil, fmt.Errorf("module zip contains invalid path %q", file.Name)
		}
		reader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file.Name, err)
		}
		data, err := io.ReadAll(io.LimitReader(reader, remaining+1))
		reader.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file.Name, err)
		}
		remaining -= int64(len(data))
		if remaining < 0 {
			return nil, errors.New("the module template is too large")
		}
		files["_template/"+name] = &fstest.MapFile{Data: data, Mode: 0o644}
	}
	if len(files) == 0 {
		return nil, errors.New("the module contains no goilerplate template")
	}
	return files, nil
}
