package project

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreatePublishesGitRepositoryWithInitialCommit(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	archive := testArchive(t, map[string]string{"go.mod": "module example.com/project"})
	if err := Create(context.Background(), bytes.NewReader(archive), destination); err != nil {
		t.Fatal(err)
	}
	for arguments, want := range map[string]string{
		"branch --show-current":        "main",
		"log -1 --format=%s":           "Initial commit",
		"log -1 --format=%an%x20<%ae>": "goilerplate <updates@goilerplate.com>",
		"status --porcelain":           "",
	} {
		fields := strings.Fields(arguments)
		command := exec.Command("git", append([]string{"-C", destination}, fields...)...)
		output, err := command.CombinedOutput()
		if err != nil || strings.TrimSpace(string(output)) != want {
			t.Fatalf("git %s = %q, %v", arguments, output, err)
		}
	}
}

func TestExtractPublishesProjectIntoMissingOrEmptyDestination(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "empty"}[existing], func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "project")
			if existing {
				if err := os.Mkdir(destination, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			archive := testArchive(t, map[string]string{"go.mod": "module example.com/project", "cmd/app/main.go": "package main"})
			if err := Extract(bytes.NewReader(archive), destination); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(destination, "cmd", "app", "main.go"))
			if err != nil || string(content) != "package main" {
				t.Fatalf("generated file = %q, %v", content, err)
			}
		})
	}
}

func TestExtractRejectsNonEmptyDestinationWithoutChangingIt(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Extract(bytes.NewReader(testArchive(t, map[string]string{"go.mod": "new"})), destination); err == nil {
		t.Fatal("non-empty destination was accepted")
	}
	content, err := os.ReadFile(existing)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing file = %q, %v", content, err)
	}
}

func TestExtractRejectsUnsafePathsWithoutPublishing(t *testing.T) {
	for _, name := range []string{"../outside", "/absolute", `..\outside`, "nested/../outside", "C:/outside"} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "project")
			if err := Extract(bytes.NewReader(testArchive(t, map[string]string{name: "bad"})), destination); err == nil {
				t.Fatalf("unsafe path %q was accepted", name)
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("destination exists after rejected archive: %v", err)
			}
		})
	}
}

func TestExtractRejectsLinksAndDuplicateFiles(t *testing.T) {
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "target"}); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Extract(bytes.NewReader(archive.Bytes()), filepath.Join(t.TempDir(), "project")); err == nil {
		t.Fatal("symbolic link was accepted")
	}

	destination := filepath.Join(t.TempDir(), "project")
	duplicate := testArchiveEntries(t, []archiveEntry{{"same", "first"}, {"same", "second"}})
	if err := Extract(bytes.NewReader(duplicate), destination); err == nil {
		t.Fatal("duplicate file was accepted")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination exists after rejected duplicate: %v", err)
	}
}

type archiveEntry struct {
	name    string
	content string
}

func testArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	entries := make([]archiveEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, archiveEntry{name, content})
	}
	return testArchiveEntries(t, entries)
}

func testArchiveEntries(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
