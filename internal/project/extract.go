package project

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxFiles = 10_000
const maxExpandedSize = 128 << 20

func Extract(source io.Reader, destination string) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	existed, err := requireMissingOrEmpty(destination)
	if err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create destination parent: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".goilerplate-*")
	if err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := extractInto(source, staging); err != nil {
		return err
	}
	if existed {
		if err := os.Remove(destination); err != nil {
			return fmt.Errorf("replace empty destination: %w", err)
		}
	}
	if err := os.Rename(staging, destination); err != nil {
		if existed {
			_ = os.Mkdir(destination, 0o755)
		}
		return fmt.Errorf("publish generated project: %w", err)
	}
	return nil
}

func requireMissingOrEmpty(destination string) (bool, error) {
	entries, err := os.ReadDir(destination)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read destination: %w", err)
	}
	if len(entries) != 0 {
		return false, errors.New("destination must be empty")
	}
	return true, nil
}

func extractInto(source io.Reader, destination string) error {
	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("open generated archive: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var totalSize int64
	files := 0
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read generated archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("generated archive contains unsupported entry %q", header.Name)
		}
		files++
		if files > maxFiles || header.Size < 0 || totalSize > maxExpandedSize-header.Size {
			return errors.New("generated archive exceeds the extraction limit")
		}
		totalSize += header.Size
		cleanName := path.Clean(header.Name)
		nativeName := filepath.FromSlash(cleanName)
		if cleanName == "." || cleanName != header.Name || path.IsAbs(cleanName) || filepath.IsAbs(nativeName) || filepath.VolumeName(nativeName) != "" || strings.HasPrefix(cleanName, "../") || strings.ContainsAny(cleanName, `\:`) {
			return fmt.Errorf("generated archive contains unsafe path %q", header.Name)
		}
		destinationPath := filepath.Join(destination, nativeName)
		relative, err := filepath.Rel(destination, destinationPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("generated archive contains unsafe path %q", header.Name)
		}
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return fmt.Errorf("create project directory: %w", err)
		}
		file, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("create project file %s: %w", cleanName, err)
		}
		_, copyErr := io.CopyN(file, tarReader, header.Size)
		closeErr := file.Close()
		if copyErr != nil {
			return fmt.Errorf("write project file %s: %w", cleanName, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close project file %s: %w", cleanName, closeErr)
		}
	}
	if files == 0 {
		return errors.New("generated archive is empty")
	}
	return nil
}
