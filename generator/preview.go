package generator

import (
	"fmt"
	"sort"
)

// PreviewFiles returns the exact generated file paths without rendering or
// exposing any template contents.
func (t Template) PreviewFiles(config Config) ([]string, error) {
	normalized, err := config.normalized()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	files, err := t.plan(normalized)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files)+1)
	paths = append(paths, "goilerplate.lock")
	for _, file := range files {
		paths = append(paths, file.Destination)
	}
	sort.Strings(paths)
	return paths, nil
}
