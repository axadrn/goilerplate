package generator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// WriteArchive renders one project and writes a byte-deterministic tar.gz tree.
func WriteArchive(destination io.Writer, templateVersion string, config Config) error {
	template, err := CurrentTemplate(templateVersion)
	if err != nil {
		return err
	}
	return template.WriteArchive(destination, config)
}

// WriteArchive renders one project from this template version.
func (t Template) WriteArchive(destination io.Writer, config Config) error {
	root, err := os.MkdirTemp("", "goilerplate-archive-")
	if err != nil {
		return fmt.Errorf("create archive workspace: %w", err)
	}
	defer os.RemoveAll(root)

	if err := t.Generate(root, config); err != nil {
		return err
	}
	paths, err := archivePaths(root)
	if err != nil {
		return err
	}

	var archive bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&archive, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)

	for _, relative := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return fmt.Errorf("read archive file %s: %w", relative, err)
		}
		header := &tar.Header{
			Name:    relative,
			Mode:    0o644,
			Size:    int64(len(content)),
			ModTime: time.Unix(0, 0),
			Format:  tar.FormatUSTAR,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("write archive header %s: %w", relative, err)
		}
		if _, err := tarWriter.Write(content); err != nil {
			return fmt.Errorf("write archive file %s: %w", relative, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	if _, err := io.Copy(destination, &archive); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}
	return nil
}

func archivePaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk generated tree: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}
