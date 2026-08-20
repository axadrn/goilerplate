package update

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMergeCreatesUpdateBranchWithoutTouchingCurrentWorktree(t *testing.T) {
	repositoryRoot := testRepository(t, map[string]string{
		"app.txt":          "title\nkeep one\nkeep two\nkeep three\ncustom: default\nfooter\n",
		"goilerplate.lock": "old lock\n",
		"removed.txt":      "remove me\n",
	})
	writeFiles(t, repositoryRoot, map[string]string{
		"app.txt":    "title\nkeep one\nkeep two\nkeep three\ncustom: customer\nfooter\n",
		"custom.txt": "customer file\n",
	})
	runTestGit(t, repositoryRoot, "add", ".")
	runTestGit(t, repositoryRoot, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", "Customer changes")
	oldRoot := testTree(t, map[string]string{
		"app.txt":          "title\nkeep one\nkeep two\nkeep three\ncustom: default\nfooter\n",
		"goilerplate.lock": "old lock\n",
		"removed.txt":      "remove me\n",
	})
	newRoot := testTree(t, map[string]string{
		"app.txt":          "new title\nkeep one\nkeep two\nkeep three\ncustom: default\nfooter\n",
		"added.txt":        "new file\n",
		"goilerplate.lock": "new lock\n",
	})

	repository, err := Open(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	beforeHead := strings.TrimSpace(runTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	result, err := Merge(context.Background(), repository, oldRoot, newRoot, "v3.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch != "goilerplate-update-v3.1.0" || len(result.Conflicts) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(result.Updated, "app.txt") || !slices.Contains(result.Updated, "added.txt") || !slices.Contains(result.Updated, "removed.txt") {
		t.Fatalf("updated = %v", result.Updated)
	}
	if afterHead := strings.TrimSpace(runTestGit(t, repositoryRoot, "rev-parse", "HEAD")); afterHead != beforeHead {
		t.Fatalf("HEAD changed from %s to %s", beforeHead, afterHead)
	}
	if status := strings.TrimSpace(runTestGit(t, repositoryRoot, "status", "--porcelain")); status != "" {
		t.Fatalf("worktree changed: %s", status)
	}
	if branch := strings.TrimSpace(runTestGit(t, repositoryRoot, "branch", "--show-current")); branch != "main" {
		t.Fatalf("current branch = %q", branch)
	}
	if author := strings.TrimSpace(runTestGit(t, repositoryRoot, "show", "-s", "--format=%an <%ae>", result.Branch)); author != "goilerplate <updates@goilerplate.com>" {
		t.Fatalf("update author = %q", author)
	}
	if content := runTestGit(t, repositoryRoot, "show", result.Branch+":app.txt"); content != "new title\nkeep one\nkeep two\nkeep three\ncustom: customer\nfooter\n" {
		t.Fatalf("merged app.txt = %q", content)
	}
	if content := runTestGit(t, repositoryRoot, "show", result.Branch+":custom.txt"); content != "customer file\n" {
		t.Fatalf("custom.txt = %q", content)
	}
	command := exec.Command("git", "cat-file", "-e", result.Branch+":removed.txt")
	command.Dir = repositoryRoot
	if err := command.Run(); err == nil {
		t.Fatal("removed template file still exists")
	}
}

func TestMergeWritesGitConflictMarkersOnNewBranch(t *testing.T) {
	repositoryRoot := testRepository(t, map[string]string{"app.txt": "color: red\n", "goilerplate.lock": "old\n"})
	writeFiles(t, repositoryRoot, map[string]string{"app.txt": "color: blue\n"})
	runTestGit(t, repositoryRoot, "add", ".")
	runTestGit(t, repositoryRoot, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", "Customer color")
	oldRoot := testTree(t, map[string]string{"app.txt": "color: red\n", "goilerplate.lock": "old\n"})
	newRoot := testTree(t, map[string]string{"app.txt": "color: green\n", "goilerplate.lock": "new\n"})

	repository, err := Open(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Merge(context.Background(), repository, oldRoot, newRoot, "v3.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(result.Conflicts, []string{"app.txt"}) {
		t.Fatalf("conflicts = %v", result.Conflicts)
	}
	content := runTestGit(t, repositoryRoot, "show", result.Branch+":app.txt")
	if !strings.Contains(content, "<<<<<<<") || !strings.Contains(content, "color: blue") || !strings.Contains(content, "color: green") {
		t.Fatalf("conflict content = %q", content)
	}
	if content := runTestGit(t, repositoryRoot, "show", result.Branch+":goilerplate.lock"); content != "new\n" {
		t.Fatalf("lock = %q", content)
	}
}

func TestOpenRejectsDirtyWorktree(t *testing.T) {
	repositoryRoot := testRepository(t, map[string]string{"app.txt": "clean\n"})
	writeFiles(t, repositoryRoot, map[string]string{"dirty.txt": "dirty\n"})
	if _, err := Open(context.Background(), repositoryRoot); err == nil || !strings.Contains(err.Error(), "clean Git worktree") {
		t.Fatalf("Open() error = %v", err)
	}
}

func TestMergePreservesExistingInternalRef(t *testing.T) {
	repositoryRoot := testRepository(t, map[string]string{"app.txt": "old\n"})
	original := strings.TrimSpace(runTestGit(t, repositoryRoot, "rev-parse", "HEAD"))
	runTestGit(t, repositoryRoot, "update-ref", "refs/goilerplate/update-import", original)
	repository, err := Open(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Merge(context.Background(), repository,
		testTree(t, map[string]string{"app.txt": "old\n"}),
		testTree(t, map[string]string{"app.txt": "new\n"}),
		"v3.1.0",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch == "" {
		t.Fatal("no update branch was created")
	}
	if current := strings.TrimSpace(runTestGit(t, repositoryRoot, "rev-parse", "refs/goilerplate/update-import")); current != original {
		t.Fatalf("existing internal ref changed from %s to %s", original, current)
	}
}

func testRepository(t *testing.T, files map[string]string) string {
	t.Helper()
	root := testTree(t, files)
	runTestGit(t, root, "init", "-b", "main")
	runTestGit(t, root, "add", ".")
	runTestGit(t, root, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", "Initial project")
	return root
}

func testTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, files)
	return root
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runTestGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}
